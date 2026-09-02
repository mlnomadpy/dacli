package cloudsync

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"time"
)

var (
	commitPattern = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`)
)

var payloadTypes = []string{
	"agent_state", "approval", "budget_state", "device_registration", "event_summary",
	"gate_evidence", "installation", "policy_bundle", "project_registration", "repository",
	"role_bundle", "run_summary", "sync_cursor", "task_proposal",
}

// PayloadTypes returns the complete, stable v1 event-type registry.
func PayloadTypes() []string {
	out := append([]string(nil), payloadTypes...)
	sort.Strings(out)
	return out
}

// ValidatePayload applies the same closed metadata allowlist used by Enqueue
// and Receive. It is exported so independent contract conformance suites can
// prove schema and runtime-validator agreement without performing I/O.
func ValidatePayload(eventType string, raw []byte) error { return validatePayload(eventType, raw) }

func validatePayload(eventType string, raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidPayload, err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: payload must be an object", ErrInvalidPayload)
	}
	var err error
	switch eventType {
	case "device_registration":
		err = validateDeviceRegistration(object)
	case "project_registration":
		err = validateProjectRegistration(object)
	case "role_bundle":
		err = validateRoleBundle(object)
	case "event_summary":
		err = validateEventSummary(object)
	case "budget_state":
		err = validateBudgetState(object)
	case "agent_state":
		err = validateAgentState(object)
	case "gate_evidence":
		err = validateGateEvidence(object)
	case "installation":
		err = closed(object, []string{"installation_id", "provider", "account_id", "state", "version"}, func() error {
			return fields(object, boundedStringsField(128, "installation_id", "account_id"), enumField("provider", "github"), enumField("state", "active", "suspended", "removed"), integerRangeField(1, 2147483647, "version"))
		})
	case "repository":
		err = closed(object, []string{"repository_id", "installation_id", "owner", "name", "visibility", "version"}, func() error {
			return fields(object, boundedStringsField(128, "repository_id", "installation_id"), boundedStringsField(255, "owner", "name"), enumField("visibility", "private", "internal", "public"), integerRangeField(1, 2147483647, "version"))
		})
	case "task_proposal":
		err = closed(object, []string{"proposal_id", "title", "acceptance", "base_version", "proposed_by"}, func() error {
			if err := fields(object, boundedStringsField(128, "proposal_id", "proposed_by"), boundedStringsField(256, "title"), integerRangeField(0, 2147483647, "base_version")); err != nil {
				return err
			}
			return uniqueStringArray(object["acceptance"], 32, 512)
		})
	case "run_summary":
		err = validateRunSummary(object)
	case "approval":
		err = closed(object, []string{"approval_id", "subject_type", "subject_id", "decision", "decided_by", "base_version"}, func() error {
			return fields(object, boundedStringsField(128, "approval_id", "subject_id", "decided_by"), enumField("subject_type", "task_proposal", "run_summary", "policy_bundle"), enumField("decision", "approved", "rejected"), integerRangeField(0, 2147483647, "base_version"))
		})
	case "policy_bundle":
		err = validatePolicyBundle(object)
	case "sync_cursor":
		err = closed(object, []string{"producer_id", "acknowledged_sequence", "acknowledged_event_id", "version"}, func() error {
			if err := fields(object, boundedStringsField(128, "producer_id"), integerRangeField(0, 9007199254740991, "acknowledged_sequence"), integerRangeField(1, 2147483647, "version"), nullableBoundedStringField(128, "acknowledged_event_id")); err != nil {
				return err
			}
			return nil
		})
	default:
		err = fmt.Errorf("unsupported event_type %q", eventType)
	}
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidPayload, err)
	}
	return nil
}

func validateDeviceRegistration(object map[string]any) error {
	return closed(object, []string{"device_id", "user_id", "name", "platform", "public_key", "state", "registered_at", "version"}, func() error {
		if err := fields(object, boundedStringsField(128, "device_id", "user_id", "name"), enumField("platform", "darwin", "linux", "windows"), enumField("state", "active", "revoked"), dateField("registered_at"), integerRangeField(1, 2147483647, "version")); err != nil {
			return err
		}
		return encodedBytesField(object, "public_key", 32)
	})
}

func validateProjectRegistration(object map[string]any) error {
	return closed(object, []string{"registration_id", "workspace_id", "repository_id", "environment_id", "default_branch", "state", "version"}, func() error {
		if err := fields(object, boundedStringsField(128, "registration_id", "workspace_id"), boundedStringsField(64, "environment_id"), boundedStringsField(255, "default_branch"), nullableBoundedStringField(128, "repository_id"), enumField("state", "connected", "disconnected"), integerRangeField(1, 2147483647, "version")); err != nil {
			return err
		}
		return nil
	})
}

func validateRoleBundle(object map[string]any) error {
	return closed(object, []string{"role_id", "role_version", "state", "digest", "issued_at", "capabilities", "runtime_preferences", "policy_version", "signing_key_id", "signature"}, func() error {
		if err := fields(object, boundedStringsField(128, "role_id", "signing_key_id"), boundedStringsField(64, "role_version"), enumField("state", "draft", "released", "deprecated", "revoked"), dateField("issued_at"), integerRangeField(1, 2147483647, "policy_version")); err != nil {
			return err
		}
		if !semverPattern.MatchString(object["role_version"].(string)) || !digestPattern.MatchString(stringValue(object, "digest")) {
			return fmt.Errorf("invalid role version or digest")
		}
		if err := uniqueStringArray(object["capabilities"], 64, 128); err != nil {
			return err
		}
		preferences, ok := object["runtime_preferences"].([]any)
		if !ok || len(preferences) > 16 {
			return fmt.Errorf("invalid runtime_preferences")
		}
		for _, value := range preferences {
			preference, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("runtime preference must be an object")
			}
			if err := closed(preference, []string{"runtime", "model"}, func() error {
				return fields(preference, boundedStringsField(64, "runtime"), boundedStringsField(128, "model"))
			}); err != nil {
				return err
			}
		}
		return encodedBytesField(object, "signature", 64)
	})
}

func validateEventSummary(object map[string]any) error {
	return closed(object, []string{"summary_id", "event_kind", "object_type", "object_id", "status", "occurred_at", "version"}, func() error {
		return fields(object, boundedStringsField(128, "summary_id", "object_id"), boundedStringsField(64, "event_kind", "status"), enumField("object_type", "task", "run", "agent", "approval", "policy", "gate", "release"), dateField("occurred_at"), integerRangeField(1, 2147483647, "version"))
	})
}

func validateBudgetState(object map[string]any) error {
	return closed(object, []string{"budget_id", "scope_type", "scope_id", "period_start", "period_end", "unit", "limit", "consumed", "state", "version"}, func() error {
		if err := fields(object, boundedStringsField(128, "budget_id", "scope_id"), enumField("scope_type", "organization", "team", "project", "environment", "task"), dateField("period_start", "period_end"), enumField("unit", "tokens", "usd_micros", "provider_units"), integerRangeField(0, 9007199254740991, "limit", "consumed"), enumField("state", "available", "warning", "paused", "exhausted"), integerRangeField(1, 2147483647, "version")); err != nil {
			return err
		}
		start, _ := time.Parse(time.RFC3339, object["period_start"].(string))
		end, _ := time.Parse(time.RFC3339, object["period_end"].(string))
		if !end.After(start) {
			return fmt.Errorf("period_end must be after period_start")
		}
		return nil
	})
}

func validateAgentState(object map[string]any) error {
	return closed(object, []string{"agent_id", "task_id", "role_id", "runtime", "model", "state", "started_at", "heartbeat_at", "version"}, func() error {
		return fields(object, boundedStringsField(128, "agent_id", "task_id", "role_id", "model"), boundedStringsField(64, "runtime"), enumField("state", "starting", "running", "waiting", "blocked", "completed", "failed", "stopped"), dateField("started_at", "heartbeat_at"), integerRangeField(1, 2147483647, "version"))
	})
}

func validateGateEvidence(object map[string]any) error {
	return closed(object, []string{"evidence_id", "run_id", "gate", "result", "commit_id", "observed_at", "version"}, func() error {
		if err := fields(object, boundedStringsField(128, "evidence_id", "run_id", "gate"), enumField("result", "pass", "fail", "unverified", "skipped"), dateField("observed_at"), integerRangeField(1, 2147483647, "version")); err != nil {
			return err
		}
		if !commitPattern.MatchString(stringValue(object, "commit_id")) {
			return fmt.Errorf("invalid commit_id")
		}
		return nil
	})
}

func validateRunSummary(object map[string]any) error {
	return closed(object, []string{"run_id", "task_id", "agent_id", "status", "started_at", "finished_at", "commit_ids", "checks"}, func() error {
		if err := fields(object, boundedStringsField(128, "run_id", "task_id", "agent_id"), enumField("status", "succeeded", "failed", "refused", "blocked"), dateField("started_at", "finished_at")); err != nil {
			return err
		}
		commits, ok := object["commit_ids"].([]any)
		if !ok || len(commits) > 64 {
			return fmt.Errorf("commit_ids must be an array")
		}
		for _, value := range commits {
			text, ok := value.(string)
			if !ok || !commitPattern.MatchString(text) {
				return fmt.Errorf("invalid commit id")
			}
		}
		checks, ok := object["checks"].([]any)
		if !ok || len(checks) > 64 {
			return fmt.Errorf("checks must be an array")
		}
		for _, value := range checks {
			check, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("check must be an object")
			}
			if err := closed(check, []string{"name", "result"}, func() error {
				return fields(check, boundedStringsField(128, "name"), enumField("result", "pass", "fail", "unverified"))
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func validatePolicyBundle(object map[string]any) error {
	return closed(object, []string{"bundle_id", "version", "effective_at", "rules"}, func() error {
		if err := fields(object, boundedStringsField(128, "bundle_id"), integerRangeField(1, 2147483647, "version"), dateField("effective_at")); err != nil {
			return err
		}
		rules, ok := object["rules"].([]any)
		if !ok || len(rules) > 64 {
			return fmt.Errorf("rules must be an array")
		}
		for _, value := range rules {
			rule, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("rule must be an object")
			}
			if err := closed(rule, []string{"capability", "effect"}, func() error {
				return fields(rule, boundedStringsField(128, "capability"), enumField("effect", "allow", "deny"))
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func closed(object map[string]any, allowed []string, validate func() error) error {
	set := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		set[key] = true
		if _, ok := object[key]; !ok {
			return fmt.Errorf("missing %s", key)
		}
	}
	for key := range object {
		if !set[key] {
			return fmt.Errorf("unexpected field %s", key)
		}
	}
	return validate()
}

type fieldCheck func(map[string]any) error

func fields(object map[string]any, checks ...fieldCheck) error {
	for _, check := range checks {
		if err := check(object); err != nil {
			return err
		}
	}
	return nil
}

func boundedStringsField(maxLength int, names ...string) fieldCheck {
	return func(object map[string]any) error {
		for _, name := range names {
			value, ok := object[name].(string)
			if !ok || value == "" || len(value) > maxLength {
				return fmt.Errorf("%s must contain 1..%d bytes", name, maxLength)
			}
		}
		return nil
	}
}

func nullableBoundedStringField(maxLength int, name string) fieldCheck {
	return func(object map[string]any) error {
		if object[name] == nil {
			return nil
		}
		value, ok := object[name].(string)
		if !ok || value == "" || len(value) > maxLength {
			return fmt.Errorf("%s must be null or contain 1..%d bytes", name, maxLength)
		}
		return nil
	}
}
func enumField(name string, allowed ...string) fieldCheck {
	return func(object map[string]any) error {
		value, ok := object[name].(string)
		if !ok {
			return fmt.Errorf("%s must be a string", name)
		}
		for _, item := range allowed {
			if value == item {
				return nil
			}
		}
		return fmt.Errorf("invalid %s", name)
	}
}
func integerRangeField(minimum, maximum int64, names ...string) fieldCheck {
	return func(object map[string]any) error {
		for _, name := range names {
			value, ok := object[name].(json.Number)
			if !ok {
				return fmt.Errorf("%s must be an integer", name)
			}
			integer, err := value.Int64()
			if err != nil || integer < minimum || integer > maximum {
				return fmt.Errorf("invalid %s", name)
			}
		}
		return nil
	}
}
func dateField(names ...string) fieldCheck {
	return func(object map[string]any) error {
		for _, name := range names {
			value, ok := object[name].(string)
			if !ok {
				return fmt.Errorf("%s must be a date-time", name)
			}
			if _, err := time.Parse(time.RFC3339, value); err != nil {
				return fmt.Errorf("invalid %s", name)
			}
		}
		return nil
	}
}

func stringArray(value any, maxItems, maxLength int) error {
	items, ok := value.([]any)
	if !ok || len(items) > maxItems {
		return fmt.Errorf("invalid string array")
	}
	for _, item := range items {
		text, ok := item.(string)
		if !ok || len(text) > maxLength {
			return fmt.Errorf("invalid string array item")
		}
	}
	return nil
}

func uniqueStringArray(value any, maxItems, maxLength int) error {
	if err := stringArray(value, maxItems, maxLength); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, item := range value.([]any) {
		text := item.(string)
		if text == "" || seen[text] {
			return fmt.Errorf("string array items must be non-empty and unique")
		}
		seen[text] = true
	}
	return nil
}

func encodedBytesField(object map[string]any, name string, size int) error {
	text, ok := object[name].(string)
	if !ok {
		return fmt.Errorf("%s must be base64", name)
	}
	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil || len(raw) != size {
		return fmt.Errorf("%s must encode exactly %d bytes", name, size)
	}
	return nil
}

func stringValue(object map[string]any, name string) string {
	value, _ := object[name].(string)
	return value
}
