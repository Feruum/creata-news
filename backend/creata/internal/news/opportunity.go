package news

import "strings"

func validateOpportunity(d Doc) error {
	if e := validateOpportunityPrototypeFields(d); e != nil {
		return e
	}
	if !has([]string{"pending", "approved", "rejected"}, str(d["status"])) {
		return errStatus(400, "Invalid opportunity status")
	}
	approved := str(d["status"]) == "approved"
	for _, key := range []string{"title", "description"} {
		if !approved && d[key] == nil {
			continue
		}
		for _, lang := range []string{"ru", "kk", "en"} {
			value, ok := object(d[key])[lang].(string)
			if approved && (!ok || strings.TrimSpace(value) == "") {
				return errStatus(400, key+" is required in "+lang+" before approval")
			}
		}
	}
	if approved {
		for _, key := range []string{"organization", "type", "country"} {
			if strings.TrimSpace(str(d[key])) == "" {
				return errStatus(400, key+" is required before approval")
			}
		}
	}
	if approved || str(d["url"]) != "" {
		if _, e := canonicalURL(str(d["url"])); e != nil {
			return errStatus(400, "A valid opportunity URL is required")
		}
	}
	if approved || str(d["deadline"]) != "" {
		if parseTime(d["deadline"]).IsZero() {
			return errStatus(400, "A valid deadline is required")
		}
	}
	if d["languages"] != nil {
		if len(array(d["languages"])) == 0 {
			return errStatus(400, "Choose at least one opportunity language")
		}
		for _, l := range array(d["languages"]) {
			if !has([]string{"ru", "kk", "en"}, str(l)) {
				return errStatus(400, "Invalid opportunity language")
			}
		}
	}
	if approved && str(d["photo"]) != "" {
		rights := object(d["photoRights"])
		if strings.TrimSpace(str(rights["credit"])) == "" || strings.TrimSpace(str(rights["rights"])) == "" || boolean(d["needsRightsReview"]) {
			return errStatus(400, "Opportunity photo credit and rights are required before approval")
		}
	}
	return nil
}

// Optional prototype presentation fields share the same persisted document as
// the opportunity. Unrelated edits preserve existing media attribution metadata.
func validateOpportunityPrototypeFields(d Doc) error {
	if value, present := d["photo"]; present {
		photo, ok := value.(string)
		if !ok {
			return errStatus(400, "Opportunity photo must be a string")
		}
		if strings.ContainsAny(photo, "\\\r\n\x00") {
			return errStatus(400, "Invalid opportunity photo URL")
		}
		if photo != "" {
			if strings.HasPrefix(photo, "/") {
				if strings.HasPrefix(photo, "//") {
					return errStatus(400, "Use a site-relative or HTTP(S) photo URL")
				}
			} else if _, e := canonicalURL(photo); e != nil {
				return errStatus(400, "Use a site-relative or HTTP(S) photo URL")
			}
		}
	}
	if value, present := d["forWhom"]; present {
		audience := object(value)
		for _, lang := range []string{"ru", "kk", "en"} {
			if _, ok := audience[lang].(string); !ok {
				return errStatus(400, "Opportunity audience must contain ru, kk and en strings")
			}
		}
	}
	return nil
}
