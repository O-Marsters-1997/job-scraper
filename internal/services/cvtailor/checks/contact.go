package checks

const checkContact = "contact"

// Contact flags a CV whose email or phone sits only in the Doc header or
// footer, or nowhere, because CV parsers often skip those segments.
func Contact(d Draft) []Finding {
	if d.Contact == nil || d.Contact.InBody {
		return nil
	}
	msg := "no email or phone was found in the CV; add your contact details to the Doc body"
	if d.Contact.InHeaderFooter {
		msg = "your email or phone is only in the Doc header or footer, which CV parsers often skip; move it into the body"
	}
	return []Finding{{Check: checkContact, Severity: Info, Message: msg}}
}
