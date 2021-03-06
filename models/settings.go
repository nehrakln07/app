package models

type Setting struct {
	Id    string      `json:"-" bson:"_id"`
	Name  string      `json:"name" bson:"name"`
	Value interface{} `json:"value" bson:"value"`
}

type Link struct {
	Id   string `json:"id" bson:"id"`
	Name string `json:"name" bson:"name"`
	Url  string `json:"url" bson:"url"`
}

type SocialNetwork struct {
	Id  string `json:"id" bson:"id"`
	Key string `json:"key" bson:"key"`
	Url string `json:"url" bson:"url"`
}

var ForbiddenSettingsKeys = []string{"backgrounds", "logo"} // Keys that are managed by their own endpoint.

const SettingsCollection = "settings"
