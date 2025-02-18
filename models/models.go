package models

import "time"

type Forum struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	Title        string    `json:"title"`
	Photo        string    `json:"photo"`
	UserID       uint      `json:"user_id"`
	User         User      `json:"user" gorm:"foreignKey:UserID;references:ID"`
	CategoryID   *uint     `json:"category_id"`
	Category     Category  `json:"category" gorm:"foreignKey:CategoryID"`
	Comments     []Comment `json:"comments" gorm:"foreignKey:ForumID"`
	Tags         []Tag     `json:"tags" gorm:"many2many:forum_tags;"`
	CreatedAt    time.Time `json:"created_at"`
	RelativeTime string    `gorm:"-" json:"relative_time"`
}

type Tag struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	UsageCount int    `json:"usage_count"`
}

type User struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	Name          string         `json:"name"`
	Username      string         `gorm:"unique;not null" form:"username" json:"username"`
	Password      string         `form:"password" json:"password"`
	Profile       string         `form:"profile" json:"profile"`
	Role          string         `form:"role" json:"role"`
	Status        string         `form:"status" json:"status"`
	SuspendUntil  *time.Time     `form:"suspend_until" json:"suspend_until,omitempty"`
	Notifications []Notification `json:"notifications" gorm:"foreignKey:UserID"`
	Forums        []Forum        `json:"forums" gorm:"foreignKey:UserID"`
	CreatedAt     time.Time      `json:"created_at"`
}

type Category struct {
	ID         uint   `gorm:"primarykey" json:"id"`
	Name       string `json:"name"`
	UsageCount int    `json:"usage_count"`
}

type Notification struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `json:"user_id"`
	CommentID *uint     `json:"comment_id,omitempty"`
	ReplyID   *uint     `json:"reply_id,omitempty"`
	Content   string    `json:"content"`
	IsRead    bool      `json:"is_read"`
	ForumID   uint      `json:"forum_id"`
	CreatedAt time.Time `json:"created_at"`
	Comment   *Comment  `json:"comment,omitempty" gorm:"foreignKey:CommentID;references:ID"`
	Reply     *Reply    `json:"reply,omitempty" gorm:"foreignKey:ReplyID;references:ID"`
	Forum     *Forum    `json:"forum,omitempty" gorm:"foreignKey:ForumID;references:ID"`
}

type Comment struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	Content      string    `json:"content"`
	ForumID      uint      `json:"forum_id"`
	UserID       uint      `json:"user_id"`
	User         User      `json:"user" gorm:"foreignKey:UserID;references:ID"`
	ParentID     *uint     `json:"parent_id"`
	Parent       *Comment  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Replies      []Reply   `json:"replies,omitempty" gorm:"foreignKey:CommentID"`
	CreatedAt    time.Time `json:"created_at"`
	RelativeTime string    `gorm:"-" json:"relative_time"`
}

type Reply struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Content       string    `json:"content"`
	CommentID     uint      `json:"comment_id"`
	ParentReplyID *uint     `json:"parent_reply_id,omitempty"`
	UserID        uint      `json:"user_id"`
	User          User      `json:"user" gorm:"foreignKey:UserID;references:ID"`
	CreatedAt     time.Time `json:"created_at"`
}

type Like struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `json:"user_id"`
	ForumID   *uint     `json:"forum_id,omitempty"`
	CommentID *uint     `json:"comment_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Report struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ReporterID uint      `json:"reporter_id"`
	ReportedID uint      `json:"reported_id"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type ForumReport struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ReporterID uint      `json:"reporter_id"`
	ForumID    uint      `json:"forum_id"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}
