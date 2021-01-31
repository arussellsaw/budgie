package util

import "os"

func IsProd() bool {
	return os.Getenv("SHEETS_ENV") == "prod"
}

func BaseURL() string {
	if IsProd() {
		return os.Getenv("SHEETS_BASEURL")
	}
	return "http://localhost:8080"
}

func Project() string {
	if IsProd() {
		return "youneedaspreadsheet"
	}
	return "russellsaw"
}
