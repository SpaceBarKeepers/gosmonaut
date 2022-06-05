package legacy

type RequestContentId struct {
	ItemId int `json:"itemID"`
}
type RequestMediaIdAndFileName struct {
	ItemId   int    `json:"itemID"`
	ItemName string `json:"itemName"`
}

type User struct {
	Id       int
	Login    string
	Password []byte
	UserName string
	Email    string
	UserType string
}

type UniversalUser struct {
	Id       int
	Login    string
	Customer string
}

type Response struct {
	ResponseCode    int
	ResponseMessage string
}

type ResponsePostMediaOK struct {
	ResponseCode int
	Id           int32
	Name         string
	Src          string
	Desc         string
	Type         string
}

type MediaItem struct {
	Id   int32
	Name string
	Src  string
	Desc string
	Type string
}

type MissingParameterResponse struct {
	Response
	MissingParameters string
}
