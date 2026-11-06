package user

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestRegistrationMethodSurvivesProfileUpdates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatal(err)
	}
	u := User{Id: 9, Password: "test-hash", RegistrationMethod: "github"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	// Profile reads deliberately omit registration fields; Save must not erase them.
	var profile User
	if err := db.Omit("registration_method").First(&profile, 9).Error; err != nil {
		t.Fatal(err)
	}
	profile.RegistrationMethod = "email"
	if err := db.Omit("AuthMethods", "UserDevices").Save(&profile).Error; err != nil {
		t.Fatal(err)
	}
	var got User
	if err := db.First(&got, 9).Error; err != nil {
		t.Fatal(err)
	}
	if got.RegistrationMethod != "github" {
		t.Fatalf("registration origin changed to %q", got.RegistrationMethod)
	}
}
