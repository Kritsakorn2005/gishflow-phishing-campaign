package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// LineContact is a LINE userId collected automatically from the Official
// Account webhook when someone follows or messages the OA. These can then be
// used as recipients (their userId) without manually looking each one up.
type LineContact struct {
	Id          int64     `json:"id" gorm:"column:id;primary_key"`
	LineID      string    `json:"line_id" gorm:"column:line_id"`
	DisplayName string    `json:"display_name" gorm:"column:display_name"`
	CreatedDate time.Time `json:"created_date"`
}

// TableName specifies the database tablename for Gorm to use.
func (LineContact) TableName() string {
	return "line_contacts"
}

// SaveLineContact stores a LINE userId if it isn't already known. If the contact
// exists and a (new) display name is provided, it is updated. It's idempotent so
// repeated follow/message events don't create duplicates.
func SaveLineContact(lineID, displayName string) error {
	if lineID == "" {
		return nil
	}
	existing := LineContact{}
	err := db.Where("line_id=?", lineID).First(&existing).Error
	if err == nil {
		if displayName != "" && existing.DisplayName != displayName {
			existing.DisplayName = displayName
			return db.Save(&existing).Error
		}
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		log.Error(err)
		return err
	}
	c := LineContact{
		LineID:      lineID,
		DisplayName: displayName,
		CreatedDate: time.Now().UTC(),
	}
	return db.Save(&c).Error
}

// GetLineContacts returns all collected LINE contacts, newest first.
func GetLineContacts() ([]LineContact, error) {
	cs := []LineContact{}
	err := db.Order("created_date desc").Find(&cs).Error
	if err != nil {
		log.Error(err)
	}
	return cs, err
}

// DeleteLineContact removes a collected LINE contact by id.
func DeleteLineContact(id int64) error {
	err := db.Where("id=?", id).Delete(LineContact{}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}
