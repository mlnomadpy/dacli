package cloudsync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

var commitPattern = regexp.MustCompile(`^[0-9a-f]{7,64}$`)

func validatePayload(eventType string, raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: payload must be an object", ErrInvalidPayload)
	}
	var err error
	switch eventType {
	case "installation":
		err = closed(object, []string{"installation_id", "provider", "account_id", "state", "version"}, func() error {
			return fields(object, stringsField("installation_id", "account_id"), enumField("provider", "github"), enumField("state", "active", "suspended", "removed"), positiveIntField("version"))
		})
	case "repository":
		err = closed(object, []string{"repository_id", "installation_id", "owner", "name", "visibility", "version"}, func() error {
			return fields(object, stringsField("repository_id", "installation_id", "owner", "name"), enumField("visibility", "private", "internal", "public"), positiveIntField("version"))
		})
	case "task_proposal":
		err = closed(object, []string{"proposal_id", "title", "acceptance", "base_version", "proposed_by"}, func() error {
			if err := fields(object, stringsField("proposal_id", "title", "proposed_by"), nonnegativeIntField("base_version")); err != nil {
				return err
			}
			if len(object["title"].(string)) > 256 {
				return fmt.Errorf("title exceeds 256 characters")
			}
			return stringArray(object["acceptance"], 32, 512)
		})
	case "run_summary":
		err = validateRunSummary(object)
	case "approval":
		err = closed(object, []string{"approval_id", "subject_type", "subject_id", "decision", "decided_by", "base_version"}, func() error {
			return fields(object, stringsField("approval_id", "subject_id", "decided_by"), enumField("subject_type", "task_proposal", "run_summary", "policy_bundle"), enumField("decision", "approved", "rejected"), nonnegativeIntField("base_version"))
		})
	case "policy_bundle":
		err = validatePolicyBundle(object)
	case "sync_cursor":
		err = closed(object, []string{"producer_id", "acknowledged_sequence", "acknowledged_event_id", "version"}, func() error {
			if err := fields(object, stringsField("producer_id"), nonnegativeIntField("acknowledged_sequence"), positiveIntField("version")); err != nil {
				return err
			}
			if value := object["acknowledged_event_id"]; value != nil {
				if _, ok := value.(string); !ok {
					return fmt.Errorf("acknowledged_event_id must be string or null")
				}
			}
			return nil
		})
	default:
		err = fmt.Errorf("unsupported event_type %q", eventType)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return nil
}

func validateRunSummary(object map[string]any) error {
	return closed(object, []string{"run_id", "task_id", "agent_id", "status", "started_at", "finished_at", "commit_ids", "checks"}, func() error {
		if err := fields(object, stringsField("run_id", "task_id", "agent_id"), enumField("status", "succeeded", "failed", "refused", "blocked"), dateField("started_at", "finished_at")); err != nil {
			return err
		}
		commits, ok := object["commit_ids"].([]any)
		if !ok {
			return fmt.Errorf("commit_ids must be an array")
		}
		for _, value := range commits {
			text, ok := value.(string)
			if !ok || !commitPattern.MatchString(text) {
				return fmt.Errorf("invalid commit id")
			}
		}
		checks, ok := object["checks"].([]any)
		if !ok {
			return fmt.Errorf("checks must be an array")
		}
		for _, value := range checks {
			check, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("check must be an object")
			}
			if err := closed(check, []string{"name", "result"}, func() error {
				return fields(check, stringsField("name"), enumField("result", "pass", "fail", "unverified"))
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func validatePolicyBundle(object map[string]any) error {
	return closed(object, []string{"bundle_id", "version", "effective_at", "rules"}, func() error {
		if err := fields(object, stringsField("bundle_id"), positiveIntField("version"), dateField("effective_at")); err != nil {
			return err
		}
		rules, ok := object["rules"].([]any)
		if !ok {
			return fmt.Errorf("rules must be an array")
		}
		for _, value := range rules {
			rule, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("rule must be an object")
			}
			if err := closed(rule, []string{"capability", "effect"}, func() error { return fields(rule, stringsField("capability"), enumField("effect", "allow", "deny")) }); err != nil {
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

func stringsField(names ...string) fieldCheck {
	return func(object map[string]any) error {
		for _, name := range names {
			value, ok := object[name].(string)
			if !ok || value == "" {
				return fmt.Errorf("%s must be a non-empty string", name)
			}
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
func positiveIntField(names ...string) fieldCheck    { return integerField(1, names...) }
func nonnegativeIntField(names ...string) fieldCheck { return integerField(0, names...) }
func integerField(minimum int64, names ...string) fieldCheck {
	return func(object map[string]any) error {
		for _, name := range names {
			value, ok := object[name].(json.Number)
			if !ok {
				return fmt.Errorf("%s must be an integer", name)
			}
			integer, err := value.Int64()
			if err != nil || integer < minimum {
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
