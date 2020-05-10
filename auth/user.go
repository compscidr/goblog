package auth

//BlogUser specifies the fields we will use to map a github identity to users
//in the blog. The only really important one is the admin, otherwise they're
//just used for comments at the moment.
//eventually we'll support other services, so we want to make sure we're
//explicit about the ID from the other system so we can map our internal ID to
//all of the other system IDs. I've used too many systems where this is broken
type BlogUser struct {
	ID          string `gorm:"primary_key"`
	GithubID    string `json:"id"`
	Login       string `json:"login"`
	AvatarURL   string `json:"avatar_url"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AccessToken string `json:"access_token"`
}
