package validation

type Validator interface {
	ValidateUsername(username string) error
	ValidateAge(age int) error
	ValidateGender(gender string) error
	ValidateCountry(country string) error
	ValidateLanguage(language string) error
	ValidateInterests(interests []string) error
}
