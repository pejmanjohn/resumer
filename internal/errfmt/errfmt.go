package errfmt

func Human(err error) string {
	if err == nil {
		return ""
	}
	return "resumer: " + err.Error()
}
