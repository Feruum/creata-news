package registrationforms

import (
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"math"
	"testing"
)

func TestVisibilityStripsHiddenChains(t *testing.T) {
	f := &pb.RegistrationForm{ApprovalMode: pb.ApprovalMode_APPROVAL_MODE_MANUAL, Fields: []*pb.FormField{
		{Id: "a", Label: "Choice", Type: pb.FormFieldType_FORM_FIELD_TYPE_SINGLE_CHOICE, Options: []*pb.FormOption{{Id: "yes", Label: "Yes"}, {Id: "no", Label: "No"}}},
		{Id: "b", Label: "Extra", Type: pb.FormFieldType_FORM_FIELD_TYPE_CHECKBOX, Required: true, Visibility: &pb.VisibilityRule{FieldId: "a", OptionId: "yes"}},
		{Id: "c", Label: "Details", Type: pb.FormFieldType_FORM_FIELD_TYPE_SHORT_TEXT, Required: true, Visibility: &pb.VisibilityRule{FieldId: "b", Checked: true}},
	}}
	if err := ValidateDefinition(f, false); err != nil {
		t.Fatal(err)
	}
	answers, err := ValidateAnswers(f, []*pb.RegistrationAnswer{{FieldId: "a", Value: &pb.RegistrationAnswer_Selection{Selection: &pb.ChoiceSelection{OptionIds: []string{"no"}}}}, {FieldId: "b", Value: &pb.RegistrationAnswer_Checked{Checked: true}}, {FieldId: "c", Value: &pb.RegistrationAnswer_Text{Text: "hidden"}}})
	if err != nil || len(answers) != 1 {
		t.Fatalf("hidden chain survived: %v %v", answers, err)
	}
}

func TestRejectInvalidDefinitionsAndAnswers(t *testing.T) {
	f := &pb.RegistrationForm{Fields: []*pb.FormField{{Id: "a", Label: "Number", Type: pb.FormFieldType_FORM_FIELD_TYPE_NUMBER, Required: true}}}
	for _, answers := range [][]*pb.RegistrationAnswer{
		nil, {{FieldId: "unknown", Value: &pb.RegistrationAnswer_Text{Text: "x"}}},
		{{FieldId: "a", Value: &pb.RegistrationAnswer_Number{Number: math.NaN()}}},
		{{FieldId: "a", Value: &pb.RegistrationAnswer_Text{Text: "7"}}},
		{{FieldId: "a", Value: &pb.RegistrationAnswer_Number{Number: 1}}, {FieldId: "a", Value: &pb.RegistrationAnswer_Number{Number: 2}}},
	} {
		if _, err := ValidateAnswers(f, answers); err == nil {
			t.Fatalf("accepted invalid answers: %v", answers)
		}
	}
	f.Fields[0].Visibility = &pb.VisibilityRule{FieldId: "a", Checked: true}
	if ValidateDefinition(f, false) == nil {
		t.Fatal("accepted cyclic visibility")
	}
}

func TestReviewTransitions(t *testing.T) {
	cases := []struct {
		status string
		action pb.RegistrationReviewAction
		next   string
	}{
		{"pending_approval", pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_APPROVE, "registered"},
		{"pending_approval", pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_REJECT, "rejected"},
		{"rejected", pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_REOPEN, "pending_approval"},
	}
	for _, c := range cases {
		got, err := reviewTransition(c.status, c.action)
		if err != nil || got != c.next {
			t.Fatalf("%s: %s %v", c.status, got, err)
		}
	}
	if _, err := reviewTransition("registered", pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_APPROVE); err == nil {
		t.Fatal("approved confirmed response")
	}
	if csvSafe("=HYPERLINK(1)") != "'=HYPERLINK(1)" {
		t.Fatal("formula injection")
	}
}

func TestNegatedBlankChoiceNeverRevealsRequiredDependent(t *testing.T) {
	f := &pb.RegistrationForm{Fields: []*pb.FormField{
		{Id: "choice", Label: "Choice", Type: pb.FormFieldType_FORM_FIELD_TYPE_SINGLE_CHOICE, Options: []*pb.FormOption{{Id: "yes", Label: "Yes"}}},
		{Id: "detail", Label: "Detail", Type: pb.FormFieldType_FORM_FIELD_TYPE_SHORT_TEXT, Required: true, Visibility: &pb.VisibilityRule{FieldId: "choice", OptionId: "yes", Negate: true}},
	}}
	for _, answers := range [][]*pb.RegistrationAnswer{nil, {{FieldId: "choice", Value: &pb.RegistrationAnswer_Selection{Selection: &pb.ChoiceSelection{}}}}} {
		if _, err := ValidateAnswers(f, answers); err != nil {
			t.Fatalf("blank answer revealed negated dependent: %v", err)
		}
	}
}

func TestFileQuestionRulesBoundFormatsAndAvailability(t *testing.T) {
	f := &pb.RegistrationForm{Fields: []*pb.FormField{{Id: "file", Label: "File", Type: pb.FormFieldType_FORM_FIELD_TYPE_FILE, FileRules: &pb.FormFileRules{Extensions: []string{"pdf"}, MaxBytes: 10 * 1024 * 1024, MaxFiles: 3}}}}
	if ValidateDefinition(f, true) == nil {
		t.Fatal("published files without private storage")
	}
	f.FilesAvailable = true
	if err := ValidateDefinition(f, true); err != nil {
		t.Fatal(err)
	}
	f.Fields[0].FileRules.Extensions = []string{"exe"}
	if ValidateDefinition(f, true) == nil {
		t.Fatal("accepted executable file type")
	}
	f.Fields[0].FileRules.Extensions = []string{"pdf"}
	f.Fields[0].FileRules.MaxFiles = 4
	if ValidateDefinition(f, true) == nil {
		t.Fatal("accepted more than three files")
	}
}

func TestAnswerKindsChoicesAndBounds(t *testing.T) {
	f := &pb.RegistrationForm{Fields: []*pb.FormField{{Id: "choice", Label: "Choice", Type: pb.FormFieldType_FORM_FIELD_TYPE_SINGLE_CHOICE, Options: []*pb.FormOption{{Id: "one", Label: "One"}, {Id: "two", Label: "Two"}}}}}
	for _, ids := range [][]string{{"missing"}, {"one", "one"}, {"one", "two"}} {
		if _, err := ValidateAnswers(f, []*pb.RegistrationAnswer{{FieldId: "choice", Value: &pb.RegistrationAnswer_Selection{Selection: &pb.ChoiceSelection{OptionIds: ids}}}}); err == nil {
			t.Fatalf("accepted invalid choices %v", ids)
		}
	}
	f.Fields[0] = &pb.FormField{Id: "email", Label: "Email", Type: pb.FormFieldType_FORM_FIELD_TYPE_EMAIL}
	if _, err := ValidateAnswers(f, []*pb.RegistrationAnswer{{FieldId: "email", Value: &pb.RegistrationAnswer_Text{Text: "Name <a@b.test>"}}}); err == nil {
		t.Fatal("accepted display-name email")
	}
}
