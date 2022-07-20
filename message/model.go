package message

type Users struct {
	UserID		string `gorm:"primaryKey"`
	UserName	string
	ConversationID	string
	FirstActive	string
}
