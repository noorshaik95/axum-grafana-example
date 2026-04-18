package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"slate/libs/common-go/tracing"
	"slate/services/tenant-service/internal/docker"
	"slate/services/tenant-service/internal/kafka"
	"slate/services/tenant-service/internal/models"
	"slate/services/tenant-service/internal/repository"
	"slate/services/tenant-service/internal/traefik"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
)

// TenantHandler handles REST API requests for tenant management.
type TenantHandler struct {
	repo        repository.TenantCRUDRepository
	provisioner *docker.TenantProvisioner
	traefikGen  *traefik.Generator
	producer    *kafka.Producer
}

// NewTenantHandler creates a TenantHandler.
func NewTenantHandler(
	repo repository.TenantCRUDRepository,
	provisioner *docker.TenantProvisioner,
	traefikGen *traefik.Generator,
	producer *kafka.Producer,
) *TenantHandler {
	return &TenantHandler{
		repo:        repo,
		provisioner: provisioner,
		traefikGen:  traefikGen,
		producer:    producer,
	}
}

// RegisterRoutes registers all tenant REST API routes on the given mux.
func (h *TenantHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /tenants", h.CreateTenant)
	mux.HandleFunc("GET /tenants", h.ListTenants)
	mux.HandleFunc("GET /tenants/{id}", h.GetTenant)
	mux.HandleFunc("PATCH /tenants/{id}/access", h.UpdateAccess)
	mux.HandleFunc("PUT /tenants/{id}/plan", h.UpdatePlan)
	mux.HandleFunc("GET /tenants/{id}/usage", h.GetUsage)
	mux.HandleFunc("DELETE /tenants/{id}", h.DeleteTenant)

	// Admin schools endpoints (alias tenants with richer data)
	mux.HandleFunc("GET /admin/schools", h.ListSchools)
	mux.HandleFunc("GET /admin/schools/{id}", h.GetSchool)

	// Admin incidents
	incidentHandler := NewIncidentHandler()
	incidentHandler.RegisterIncidentRoutes(mux)

	// Admin feature flags
	flagsHandler := NewFlagsHandler()
	flagsHandler.RegisterFlagsRoutes(mux)
}

// ListSchools handles GET /admin/schools — tenants with health + renewal metadata.
func (h *TenantHandler) ListSchools(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	search := r.URL.Query().Get("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	tenants, total, err := h.repo.ListTenants(r.Context(), page, pageSize, search)
	if err != nil {
		log.Error().Err(err).Msg("failed to list schools")
		writeError(w, http.StatusInternalServerError, "failed to list schools")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"schools":  tenants,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetSchool handles GET /admin/schools/:id — detailed school view with incident data.
func (h *TenantHandler) GetSchool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tenant, err := h.repo.GetTenantByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "school not found")
		return
	}

	// Augment with mock health/renewal data
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"school":         tenant,
		"health":         "healthy",
		"renewalDate":    nil,
		"openIncidents":  0,
		"activeStudents": 0,
	})
}

func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.StartSpan(r.Context(), "handler.CreateTenant")
	defer span.End()
	r = r.WithContext(ctx)

	var req models.ProvisionTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Slug == "" || req.AdminEmail == "" {
		writeError(w, http.StatusBadRequest, "name, slug, and adminEmail are required")
		return
	}

	// Check for duplicate slug
	if existing, _ := h.repo.GetBySlug(r.Context(), req.Slug); existing != nil {
		writeError(w, http.StatusConflict, "tenant with this slug already exists")
		return
	}

	plan := models.Plan{
		StorageGB:  10,
		MaxUsers:   50,
		MaxCourses: 25,
		Features:   []string{"api_access", "basic_analytics"},
	}
	if req.Plan != nil {
		plan = *req.Plan
	}

	tenant := &models.TenantV2{
		Slug:       req.Slug,
		Name:       req.Name,
		AdminEmail: req.AdminEmail,
		Status:     models.StatusProvisioning,
		Plan:       plan,
		Subdomain:  req.Slug + ".slate.local",
	}

	if err := h.repo.CreateTenant(r.Context(), tenant); err != nil {
		log.Error().Err(err).Msg("failed to create tenant")
		writeError(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}

	// Write Traefik config
	if err := h.traefikGen.WriteConfig(tenant.ID, tenant.Slug); err != nil {
		log.Error().Err(err).Msg("failed to write traefik config")
	}

	// Provision containers asynchronously
	go func() {
		ctx := r.Context()
		containerIDs, err := h.provisioner.Provision(ctx, tenant)
		if err != nil {
			log.Error().Err(err).Str("tenantId", tenant.ID).Msg("docker provisioning failed")
			h.repo.UpdateTenantStatus(ctx, tenant.ID, "failed")
			return
		}

		tenant.ContainerIDs = containerIDs
		tenant.Status = models.StatusActive
		h.repo.UpdateTenantAfterProvision(ctx, tenant)

		h.producer.ProduceTenantProvisioned(ctx, &kafka.TenantProvisionedEvent{
			TenantID:     tenant.ID,
			Slug:         tenant.Slug,
			Subdomain:    tenant.Subdomain,
			ContainerIDs: containerIDs,
		})
	}()

	writeJSON(w, http.StatusCreated, tenant)
}

func (h *TenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	search := r.URL.Query().Get("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	tenants, total, err := h.repo.ListTenants(r.Context(), page, pageSize, search)
	if err != nil {
		log.Error().Err(err).Msg("failed to list tenants")
		writeError(w, http.StatusInternalServerError, "failed to list tenants")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tenants":  tenants,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *TenantHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.StartSpan(r.Context(), "handler.GetTenant",
		attribute.String("tenant.id", r.PathValue("id")))
	defer span.End()

	id := r.PathValue("id")
	tenant, err := h.repo.GetTenantByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	writeJSON(w, http.StatusOK, tenant)
}

func (h *TenantHandler) UpdateAccess(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req models.UpdateAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenant, err := h.repo.GetTenantByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	if req.Enabled {
		// Re-enable: start containers
		if err := h.provisioner.StartContainers(r.Context(), id); err != nil {
			log.Error().Err(err).Msg("failed to start containers")
			writeError(w, http.StatusInternalServerError, "failed to enable tenant")
			return
		}
		if err := h.repo.UpdateTenantStatus(r.Context(), id, models.StatusActive); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update status")
			return
		}
		h.producer.ProduceTenantEnabled(r.Context(), id)
		tenant.Status = models.StatusActive
	} else {
		// Disable: stop containers
		if err := h.provisioner.StopContainers(r.Context(), id); err != nil {
			log.Error().Err(err).Msg("failed to stop containers")
			writeError(w, http.StatusInternalServerError, "failed to disable tenant")
			return
		}
		if err := h.repo.UpdateTenantStatus(r.Context(), id, models.StatusSuspended); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update status")
			return
		}
		h.producer.ProduceTenantDisabled(r.Context(), id)
		tenant.Status = models.StatusSuspended
	}

	log.Info().Str("tenantId", id).Bool("enabled", req.Enabled).Str("reason", req.Reason).Msg("tenant access updated")
	writeJSON(w, http.StatusOK, tenant)
}

func (h *TenantHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req models.UpdatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tenant, err := h.repo.GetTenantByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	plan := tenant.Plan
	if req.StorageGB != nil {
		plan.StorageGB = *req.StorageGB
	}
	if req.MaxUsers != nil {
		plan.MaxUsers = *req.MaxUsers
	}
	if req.MaxCourses != nil {
		plan.MaxCourses = *req.MaxCourses
	}
	if req.Features != nil {
		plan.Features = *req.Features
	}

	if err := h.repo.UpdateTenantPlan(r.Context(), id, plan); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update plan")
		return
	}

	h.producer.ProduceTenantPlanUpdated(r.Context(), id, plan)

	tenant.Plan = plan
	writeJSON(w, http.StatusOK, tenant)
}

func (h *TenantHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	tenant, err := h.repo.GetTenantByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	usage := models.TenantUsage{
		TenantID:       tenant.ID,
		StorageUsedGB:  0, // Would query actual usage from storage service
		StorageLimitGB: tenant.Plan.StorageGB,
		UserCount:      0,
		MaxUsers:       tenant.Plan.MaxUsers,
		CourseCount:    0,
		MaxCourses:     tenant.Plan.MaxCourses,
		ContainerCount: len(tenant.ContainerIDs),
	}

	writeJSON(w, http.StatusOK, usage)
}

func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.StartSpan(r.Context(), "handler.DeleteTenant",
		attribute.String("tenant.id", r.PathValue("id")))
	defer span.End()

	id := r.PathValue("id")

	tenant, err := h.repo.GetTenantByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	// Deprovision containers
	if err := h.provisioner.Deprovision(ctx, id); err != nil {
		log.Error().Err(err).Msg("failed to deprovision containers")
	}

	// Delete Traefik config
	if err := h.traefikGen.DeleteConfig(id); err != nil {
		log.Error().Err(err).Msg("failed to delete traefik config")
	}

	// Soft delete
	if err := h.repo.SoftDeleteTenant(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete tenant")
		return
	}

	_ = tenant
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "tenantId": id})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
