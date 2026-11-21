package registrationforms

import (
	"errors"
	"fmt"
	"math"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"google.golang.org/protobuf/proto"
)

var stableID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,100}$`)
var phonePattern = regexp.MustCompile(`^[+0-9() .-]{5,40}$`)

func invalid(message string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(message))
}
func precondition(message string) error {
	return connect.NewError(connect.CodeFailedPrecondition, errors.New(message))
}

// ValidateDefinition bounds definitions and makes visibility a directed acyclic
// graph by allowing dependencies only on earlier choice or checkbox fields.
func ValidateDefinition(f *pb.RegistrationForm, publishing bool) error {
	if f == nil {
		return invalid("form is required")
	}
	if len(f.Fields) > 50 {
		return invalid("a form may contain at most 50 questions")
	}
	if f.ApprovalMode != pb.ApprovalMode_APPROVAL_MODE_UNSPECIFIED && f.ApprovalMode != pb.ApprovalMode_APPROVAL_MODE_AUTOMATIC && f.ApprovalMode != pb.ApprovalMode_APPROVAL_MODE_MANUAL {
		return invalid("unsupported approval mode")
	}
	earlier := map[string]*pb.FormField{}
	for _, field := range f.Fields {
		if field == nil || !stableID.MatchString(field.Id) || earlier[field.Id] != nil {
			return invalid("question IDs must be unique stable identifiers")
		}
		if strings.TrimSpace(field.Label) == "" || utf8.RuneCountInString(field.Label) > 300 || utf8.RuneCountInString(field.Description) > 2000 {
			return invalid("question label or description exceeds its bounds")
		}
		if field.Type < pb.FormFieldType_FORM_FIELD_TYPE_SHORT_TEXT || field.Type > pb.FormFieldType_FORM_FIELD_TYPE_FILE {
			return invalid("unsupported question type")
		}
		choice := field.Type == pb.FormFieldType_FORM_FIELD_TYPE_SINGLE_CHOICE || field.Type == pb.FormFieldType_FORM_FIELD_TYPE_MULTIPLE_CHOICE
		ids := map[string]bool{}
		if choice && (len(field.Options) < 1 || len(field.Options) > 100) {
			return invalid("choice questions require 1 to 100 options")
		}
		if !choice && len(field.Options) > 0 {
			return invalid("only choice questions may have options")
		}
		for _, option := range field.Options {
			if option == nil || !stableID.MatchString(option.Id) || ids[option.Id] || strings.TrimSpace(option.Label) == "" || utf8.RuneCountInString(option.Label) > 300 {
				return invalid("invalid or duplicate choice option")
			}
			ids[option.Id] = true
		}
		if rule := field.Visibility; rule != nil {
			parent := earlier[rule.FieldId]
			if parent == nil {
				return invalid("visibility must reference an earlier question")
			}
			switch parent.Type {
			case pb.FormFieldType_FORM_FIELD_TYPE_CHECKBOX:
				if rule.OptionId != "" {
					return invalid("checkbox visibility cannot reference an option")
				}
			case pb.FormFieldType_FORM_FIELD_TYPE_SINGLE_CHOICE, pb.FormFieldType_FORM_FIELD_TYPE_MULTIPLE_CHOICE:
				found := false
				for _, option := range parent.Options {
					found = found || option.Id == rule.OptionId
				}
				if !found {
					return invalid("visibility references an unknown option")
				}
			default:
				return invalid("visibility must reference a choice or checkbox")
			}
		}
		if field.Type == pb.FormFieldType_FORM_FIELD_TYPE_FILE {
			if publishing && !f.FilesAvailable {
				return precondition("file storage is unavailable; remove file questions or configure private storage")
			}
			rules := field.FileRules
			if rules == nil || len(rules.Extensions) == 0 || rules.MaxBytes <= 0 || rules.MaxBytes > 10*1024*1024 || rules.MaxFiles < 1 || rules.MaxFiles > 3 {
				return invalid("file questions require allowed formats, at most 10 MiB per file and 1 to 3 files")
			}
			for _, ext := range rules.Extensions {
				switch ext {
				case "pdf", "jpg", "jpeg", "png", "webp", "docx", "xlsx":
				default:
					return invalid("unsupported file format")
				}
			}
		} else if field.FileRules != nil {
			return invalid("only file questions may have file rules")
		}
		earlier[field.Id] = field
	}
	return nil
}

func visible(rule *pb.VisibilityRule, values map[string]*pb.RegistrationAnswer) bool {
	if rule == nil {
		return true
	}
	a := values[rule.FieldId]
	if a == nil {
		return false
	}
	matched := false
	switch value := a.Value.(type) {
	case *pb.RegistrationAnswer_Checked:
		matched = value.Checked == rule.Checked
	case *pb.RegistrationAnswer_Selection:
		if len(value.Selection.GetOptionIds()) == 0 {
			return false
		}
		for _, id := range value.Selection.GetOptionIds() {
			matched = matched || id == rule.OptionId
		}
	}
	if rule.Negate {
		return !matched
	}
	return matched
}

// ValidateAnswers returns normalized visible answers only. Hidden answers never
// influence descendants, required validation, persistence, or attachment claims.
func ValidateAnswers(f *pb.RegistrationForm, answers []*pb.RegistrationAnswer) ([]*pb.RegistrationAnswer, error) {
	if err := ValidateDefinition(f, false); err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, field := range f.Fields {
		known[field.Id] = true
	}
	incoming := map[string]*pb.RegistrationAnswer{}
	for _, answer := range answers {
		if answer == nil || !known[answer.FieldId] || incoming[answer.FieldId] != nil {
			return nil, invalid("unknown or duplicate question answer")
		}
		incoming[answer.FieldId] = answer
	}
	values := map[string]*pb.RegistrationAnswer{}
	result := []*pb.RegistrationAnswer{}
	for _, field := range f.Fields {
		if !visible(field.Visibility, values) {
			continue
		}
		answer := incoming[field.Id]
		if answer == nil || answer.Value == nil {
			if field.Required {
				return nil, invalid(fmt.Sprintf("%s is required", field.Label))
			}
			continue
		}
		a := proto.Clone(answer).(*pb.RegistrationAnswer)
		empty := false
		switch field.Type {
		case pb.FormFieldType_FORM_FIELD_TYPE_SHORT_TEXT, pb.FormFieldType_FORM_FIELD_TYPE_LONG_TEXT, pb.FormFieldType_FORM_FIELD_TYPE_EMAIL, pb.FormFieldType_FORM_FIELD_TYPE_PHONE:
			v, ok := a.Value.(*pb.RegistrationAnswer_Text)
			if !ok {
				return nil, invalid("text answer required")
			}
			v.Text = strings.TrimSpace(v.Text)
			empty = v.Text == ""
			max := 1000
			if field.Type == pb.FormFieldType_FORM_FIELD_TYPE_LONG_TEXT {
				max = 10000
			}
			if utf8.RuneCountInString(v.Text) > max {
				return nil, invalid("answer is too long")
			}
			if !empty && field.Type == pb.FormFieldType_FORM_FIELD_TYPE_EMAIL {
				parsed, err := mail.ParseAddress(v.Text)
				if err != nil || parsed.Address != v.Text || len(v.Text) > 254 {
					return nil, invalid("invalid email answer")
				}
			}
			if !empty && field.Type == pb.FormFieldType_FORM_FIELD_TYPE_PHONE && !phonePattern.MatchString(v.Text) {
				return nil, invalid("invalid phone answer")
			}
		case pb.FormFieldType_FORM_FIELD_TYPE_NUMBER:
			v, ok := a.Value.(*pb.RegistrationAnswer_Number)
			if !ok || math.IsNaN(v.Number) || math.IsInf(v.Number, 0) {
				return nil, invalid("finite numeric answer required")
			}
		case pb.FormFieldType_FORM_FIELD_TYPE_CHECKBOX:
			v, ok := a.Value.(*pb.RegistrationAnswer_Checked)
			if !ok {
				return nil, invalid("checkbox answer required")
			}
			empty = !v.Checked
		case pb.FormFieldType_FORM_FIELD_TYPE_SINGLE_CHOICE, pb.FormFieldType_FORM_FIELD_TYPE_MULTIPLE_CHOICE:
			v, ok := a.Value.(*pb.RegistrationAnswer_Selection)
			if !ok || v.Selection == nil {
				return nil, invalid("choice answer required")
			}
			ids := v.Selection.OptionIds
			empty = len(ids) == 0
			if field.Type == pb.FormFieldType_FORM_FIELD_TYPE_SINGLE_CHOICE && len(ids) > 1 {
				return nil, invalid("select one option")
			}
			allowed := map[string]bool{}
			for _, o := range field.Options {
				allowed[o.Id] = true
			}
			seen := map[string]bool{}
			for _, id := range ids {
				if !allowed[id] || seen[id] {
					return nil, invalid("unknown or duplicate option")
				}
				seen[id] = true
			}
		case pb.FormFieldType_FORM_FIELD_TYPE_FILE:
			v, ok := a.Value.(*pb.RegistrationAnswer_Files)
			if !ok || v.Files == nil {
				return nil, invalid("file answer required")
			}
			empty = len(v.Files.UploadIds) == 0
			if len(v.Files.UploadIds) > int(field.FileRules.MaxFiles) {
				return nil, invalid("too many files")
			}
			seen := map[string]bool{}
			for _, id := range v.Files.UploadIds {
				if id == "" || seen[id] {
					return nil, invalid("invalid or duplicate upload")
				}
				seen[id] = true
			}
		default:
			return nil, invalid("unsupported question type")
		}
		if empty && field.Required {
			return nil, invalid(fmt.Sprintf("%s is required", field.Label))
		}
		values[field.Id] = a
		result = append(result, a)
	}
	return result, nil
}

func reviewTransition(status string, action pb.RegistrationReviewAction) (string, error) {
	switch {
	case status == "pending_approval" && action == pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_APPROVE:
		return "registered", nil
	case status == "pending_approval" && action == pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_REJECT:
		return "rejected", nil
	case status == "rejected" && action == pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_REOPEN:
		return "pending_approval", nil
	}
	return "", precondition("registration status changed; reload before reviewing")
}
func csvSafe(value string) string {
	trim := strings.TrimLeft(value, " \t\r\n")
	if len(trim) > 0 && strings.ContainsRune("=+-@", rune(trim[0])) {
		return "'" + value
	}
	return value
}
