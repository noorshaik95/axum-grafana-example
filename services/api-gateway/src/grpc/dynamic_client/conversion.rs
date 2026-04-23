//! JSON to Protobuf conversion using prost-reflect.
//!
//! Handles bidirectional conversion between JSON and Protocol Buffer formats
//! using dynamic message types.

use prost::Message;
use prost_reflect::{DescriptorPool, DeserializeOptions, DynamicMessage};
use tracing::{debug, error};

use crate::grpc::types::GrpcError;

/// Convert JSON payload to protobuf bytes using prost-reflect.
///
/// # Arguments
///
/// * `json_bytes` - The JSON payload as bytes
/// * `method` - The method name to find the input type
/// * `pool` - The descriptor pool containing type information
/// * `service_name` - The fully qualified service name
///
/// # Returns
///
/// Protobuf-encoded bytes ready for gRPC transmission
pub fn json_to_protobuf(
    json_bytes: &[u8],
    method: &str,
    pool: &DescriptorPool,
    service_name: &str,
) -> Result<Vec<u8>, GrpcError> {
    // Find method descriptor
    let service_desc = pool.get_service_by_name(service_name).ok_or_else(|| {
        error!(service = %service_name, "Service not found in descriptor pool");
        GrpcError::ConversionError(format!("Service {} not found", service_name))
    })?;

    let method_desc = service_desc
        .methods()
        .find(|m| m.name() == method)
        .ok_or_else(|| {
            error!(method = %method, "Method not found in service descriptor");
            GrpcError::ConversionError(format!("Method {} not found", method))
        })?;

    let input_desc = method_desc.input();

    // Parse and clean JSON
    let cleaned_json = clean_gateway_metadata(json_bytes)?;

    // Deserialize JSON into dynamic message.
    //
    // Tolerate unknown fields (T2): FE payloads frequently carry extras that
    // aren't part of the proto (client-side UI state, echoed metadata,
    // forward-compat additions). prost-reflect defaults `deny_unknown_fields=true`
    // which turns every such extra into a 502 Conversion error. Flipping to
    // `false` makes the converter drop unknown fields silently — matching
    // google.protobuf.json-format's standard permissive behaviour.
    let options = DeserializeOptions::new().deny_unknown_fields(false);
    let mut deserializer = serde_json::Deserializer::from_str(&cleaned_json);
    let message = DynamicMessage::deserialize_with_options(
        input_desc.clone(),
        &mut deserializer,
        &options,
    )
    .map_err(|e| {
        error!(error = %e, "Failed to deserialize JSON to protobuf");
        GrpcError::ConversionError(format!("JSON to protobuf conversion failed: {}", e))
    })?;

    // Ensure all JSON was consumed
    deserializer.end().map_err(|e| {
        error!(error = %e, "JSON has trailing data");
        GrpcError::ConversionError(format!("Invalid JSON: {}", e))
    })?;

    // Encode to protobuf bytes
    let mut buf = Vec::new();
    message.encode(&mut buf).map_err(|e| {
        error!(error = %e, "Failed to encode protobuf message");
        GrpcError::ConversionError(format!("Protobuf encoding failed: {}", e))
    })?;

    debug!(
        service = %service_name,
        method = %method,
        protobuf_size = buf.len(),
        "Successfully converted JSON to protobuf"
    );

    Ok(buf)
}

/// Convert protobuf bytes back to JSON using prost-reflect.
///
/// # Arguments
///
/// * `protobuf_bytes` - The protobuf-encoded response
/// * `method` - The method name to find the output type
/// * `pool` - The descriptor pool containing type information
/// * `service_name` - The fully qualified service name
///
/// # Returns
///
/// JSON-encoded bytes
pub fn protobuf_to_json(
    protobuf_bytes: &[u8],
    method: &str,
    pool: &DescriptorPool,
    service_name: &str,
) -> Result<Vec<u8>, GrpcError> {
    // Find method descriptor
    let service_desc = pool.get_service_by_name(service_name).ok_or_else(|| {
        error!(service = %service_name, "Service not found in descriptor pool");
        GrpcError::ConversionError(format!("Service {} not found", service_name))
    })?;

    let method_desc = service_desc
        .methods()
        .find(|m| m.name() == method)
        .ok_or_else(|| {
            error!(method = %method, "Method not found in service descriptor");
            GrpcError::ConversionError(format!("Method {} not found", method))
        })?;

    let output_desc = method_desc.output();

    // Decode protobuf message
    let message = DynamicMessage::decode(output_desc.clone(), protobuf_bytes).map_err(|e| {
        error!(error = %e, "Failed to decode protobuf message");
        GrpcError::ConversionError(format!("Protobuf decoding failed: {}", e))
    })?;

    // Serialize to JSON
    let json_string = serde_json::to_string(&message).map_err(|e| {
        error!(error = %e, "Failed to serialize protobuf to JSON");
        GrpcError::ConversionError(format!("Protobuf to JSON conversion failed: {}", e))
    })?;

    let json_bytes = json_string.into_bytes();

    debug!(
        service = %service_name,
        method = %method,
        json_size = json_bytes.len(),
        "Successfully converted protobuf to JSON"
    );

    Ok(json_bytes)
}

/// Clean gateway metadata fields from JSON payload.
///
/// Removes internal gateway fields that aren't part of the protobuf schema.
fn clean_gateway_metadata(json_bytes: &[u8]) -> Result<String, GrpcError> {
    let mut json_value: serde_json::Value = serde_json::from_slice(json_bytes).map_err(|e| {
        error!(error = %e, "JSON payload is not valid JSON");
        GrpcError::ConversionError(format!("Invalid JSON: {}", e))
    })?;

    // Remove gateway metadata fields
    if let Some(obj) = json_value.as_object_mut() {
        obj.remove("_auth_user_id");
        obj.remove("_auth_roles");
        obj.remove("_trace");
    }

    // Convert back to JSON string
    serde_json::to_string(&json_value).map_err(|e| {
        error!(error = %e, "Failed to serialize cleaned JSON");
        GrpcError::ConversionError(format!("JSON serialization failed: {}", e))
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Build a descriptor pool from an inline proto FileDescriptorProto so the
    /// test can exercise the converter without the real registry. We use a
    /// minimal `Msg { field: string f = 1 }` type.
    fn tiny_pool() -> (DescriptorPool, prost_reflect::MessageDescriptor) {
        use prost_reflect::prost_types::{
            field_descriptor_proto, DescriptorProto, FieldDescriptorProto, FileDescriptorProto,
        };

        let field = FieldDescriptorProto {
            name: Some("f".to_string()),
            number: Some(1),
            r#type: Some(field_descriptor_proto::Type::String as i32),
            label: Some(field_descriptor_proto::Label::Optional as i32),
            json_name: Some("f".to_string()),
            ..Default::default()
        };
        let msg = DescriptorProto {
            name: Some("Msg".to_string()),
            field: vec![field],
            ..Default::default()
        };
        let file = FileDescriptorProto {
            name: Some("tiny.proto".to_string()),
            package: Some("tiny".to_string()),
            syntax: Some("proto3".to_string()),
            message_type: vec![msg],
            ..Default::default()
        };
        let mut pool = DescriptorPool::new();
        pool.add_file_descriptor_proto(file).unwrap();
        let desc = pool.get_message_by_name("tiny.Msg").unwrap();
        (pool, desc)
    }

    #[test]
    fn converter_drops_unknown_fields_instead_of_erroring() {
        // T2 primary bug: prost-reflect's default `deny_unknown_fields=true`
        // turned every FE payload carrying a non-proto field into a 502
        // Conversion error. The converter must now accept and drop extras.
        let (_pool, desc) = tiny_pool();
        let json = br#"{"f":"hello","unknownClientState":"ignored"}"#;
        let options = DeserializeOptions::new().deny_unknown_fields(false);
        let mut de = serde_json::Deserializer::from_slice(json);
        let msg = DynamicMessage::deserialize_with_options(desc, &mut de, &options)
            .expect("unknown fields must not error the converter");
        let got = msg
            .get_field_by_name("f")
            .expect("f present")
            .as_str()
            .unwrap()
            .to_string();
        assert_eq!(got, "hello");
    }

    #[test]
    fn converter_default_options_would_reject_unknown_fields() {
        // Guard: if this ever flips upstream, we want to know immediately
        // so our explicit override stays meaningful.
        let (_pool, desc) = tiny_pool();
        let json = br#"{"f":"hello","extra":1}"#;
        let mut de = serde_json::Deserializer::from_slice(json);
        let res = DynamicMessage::deserialize(desc, &mut de);
        assert!(
            res.is_err(),
            "upstream default was expected to deny_unknown_fields; if this passes, the override in json_to_protobuf is obsolete"
        );
    }
}
