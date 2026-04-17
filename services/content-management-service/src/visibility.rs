use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Visibility rules for content objects
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum VisibilityRule {
    Always,
    AfterDate { date: DateTime<Utc> },
    SpecificGroups { group_ids: Vec<String> },
    Hidden,
}

impl VisibilityRule {
    /// Check if content is visible to a user based on their group memberships
    pub fn is_visible_to(&self, user_groups: &[String]) -> bool {
        match self {
            VisibilityRule::Always => true,
            VisibilityRule::AfterDate { date } => Utc::now() >= *date,
            VisibilityRule::SpecificGroups { group_ids } => {
                user_groups.iter().any(|g| group_ids.contains(g))
            }
            VisibilityRule::Hidden => false,
        }
    }
}

impl Default for VisibilityRule {
    fn default() -> Self {
        VisibilityRule::Always
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Duration;

    #[test]
    fn test_always_visible() {
        let rule = VisibilityRule::Always;
        assert!(rule.is_visible_to(&[]));
        assert!(rule.is_visible_to(&["group1".to_string()]));
    }

    #[test]
    fn test_hidden() {
        let rule = VisibilityRule::Hidden;
        assert!(!rule.is_visible_to(&[]));
        assert!(!rule.is_visible_to(&["group1".to_string()]));
    }

    #[test]
    fn test_after_date_past() {
        let rule = VisibilityRule::AfterDate {
            date: Utc::now() - Duration::hours(1),
        };
        assert!(rule.is_visible_to(&[]));
    }

    #[test]
    fn test_after_date_future() {
        let rule = VisibilityRule::AfterDate {
            date: Utc::now() + Duration::hours(1),
        };
        assert!(!rule.is_visible_to(&[]));
    }

    #[test]
    fn test_specific_groups_match() {
        let rule = VisibilityRule::SpecificGroups {
            group_ids: vec!["instructors".to_string(), "ta".to_string()],
        };
        assert!(rule.is_visible_to(&["instructors".to_string()]));
        assert!(rule.is_visible_to(&["ta".to_string(), "students".to_string()]));
    }

    #[test]
    fn test_specific_groups_no_match() {
        let rule = VisibilityRule::SpecificGroups {
            group_ids: vec!["instructors".to_string()],
        };
        assert!(!rule.is_visible_to(&[]));
        assert!(!rule.is_visible_to(&["students".to_string()]));
    }

    #[test]
    fn test_serde_roundtrip() {
        let rule = VisibilityRule::AfterDate {
            date: Utc::now(),
        };
        let json = serde_json::to_value(&rule).unwrap();
        assert_eq!(json["type"], "after_date");

        let deserialized: VisibilityRule = serde_json::from_value(json).unwrap();
        assert!(matches!(deserialized, VisibilityRule::AfterDate { .. }));
    }

    #[test]
    fn test_serde_always() {
        let json = serde_json::json!({"type": "always"});
        let rule: VisibilityRule = serde_json::from_value(json).unwrap();
        assert!(matches!(rule, VisibilityRule::Always));
    }

    #[test]
    fn test_serde_hidden() {
        let json = serde_json::json!({"type": "hidden"});
        let rule: VisibilityRule = serde_json::from_value(json).unwrap();
        assert!(matches!(rule, VisibilityRule::Hidden));
    }

    #[test]
    fn test_serde_specific_groups() {
        let json = serde_json::json!({
            "type": "specific_groups",
            "group_ids": ["group1", "group2"]
        });
        let rule: VisibilityRule = serde_json::from_value(json).unwrap();
        assert!(matches!(rule, VisibilityRule::SpecificGroups { .. }));
        assert!(rule.is_visible_to(&["group1".to_string()]));
    }
}
