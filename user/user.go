package user

// User holds all of the info the chat app need
type User struct {
	ID                       string `json:"sub"`
	Name                     string `json:"name"`
	FirstName                string `json:"given_name`
	LastName                 string `json:"family_name`
	GoogleProfilePictureLink string `json:"picture"`
	Email                    string `json:"email"`
}
