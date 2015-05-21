package models

// HTTPAuth is used for HTTP Basic Auth.
type HTTPAuth struct {
	Username string `json:"username,omitempty" bson:"username,omitempty"`
	Password string `json:"password,omitempty" bson:"password,omitempty"`
}
