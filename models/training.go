package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
)

// TrainingPage contains the awareness / "teachable moment" content that is
// shown to a recipient after they interact with a phishing simulation
// (clicked a link on a no-form page, or submitted data on a landing page).
type TrainingPage struct {
	Id           int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId       int64     `json:"-" gorm:"column:user_id"`
	Name         string    `json:"name"`
	HTML         string    `json:"html" gorm:"column:html"`
	ModifiedDate time.Time `json:"modified_date"`
}

// ErrTrainingNameNotSpecified is thrown if the name of the training page is blank.
var ErrTrainingNameNotSpecified = errors.New("Training page name not specified")

// TableName specifies the database tablename for Gorm to use.
func (t TrainingPage) TableName() string {
	return "training_pages"
}

// Validate ensures that a training page contains the appropriate details.
func (t *TrainingPage) Validate() error {
	if t.Name == "" {
		return ErrTrainingNameNotSpecified
	}
	// The content supports the same phishing template variables as landing
	// pages (e.g. {{.FirstName}}), so validate it the same way.
	return ValidateTemplate(t.HTML)
}

// GetTrainingPages returns the training pages owned by the given user.
func GetTrainingPages(uid int64) ([]TrainingPage, error) {
	ts := []TrainingPage{}
	err := db.Where("user_id=?", uid).Find(&ts).Error
	if err != nil {
		log.Error(err)
		return ts, err
	}
	return ts, err
}

// GetTrainingPage returns the training page, if it exists, specified by the
// given id and user_id.
func GetTrainingPage(id int64, uid int64) (TrainingPage, error) {
	t := TrainingPage{}
	err := db.Where("user_id=? and id=?", uid, id).Find(&t).Error
	if err != nil {
		log.Error(err)
	}
	return t, err
}

// GetTrainingPageByName returns the training page, if it exists, specified by
// the given name and user_id.
func GetTrainingPageByName(n string, uid int64) (TrainingPage, error) {
	t := TrainingPage{}
	err := db.Where("user_id=? and name=?", uid, n).Find(&t).Error
	if err != nil {
		log.Error(err)
	}
	return t, err
}

// PostTrainingPage creates a new training page in the database.
func PostTrainingPage(t *TrainingPage) error {
	if err := t.Validate(); err != nil {
		log.Error(err)
		return err
	}
	err := db.Save(t).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// PutTrainingPage edits an existing training page in the database.
func PutTrainingPage(t *TrainingPage) error {
	if err := t.Validate(); err != nil {
		return err
	}
	err := db.Where("id=?", t.Id).Save(t).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// DeleteTrainingPage deletes an existing training page in the database.
func DeleteTrainingPage(id int64, uid int64) error {
	err := db.Where("user_id=?", uid).Delete(TrainingPage{Id: id}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}
