package pagemanager

import (
	"database/sql"

	"github.com/bokwoon95/pagemanager/cryptoutil"
)

type PageManager struct {
	keybox *cryptoutil.KeyBox
	pwbox  *cryptoutil.PasswordBox
	// themesFS
	// imageGetter/imageSetter
	// pagemanagerFS
	// pluginsFS
	dataDB       *sql.DB
	superadminDB *sql.DB
}

// theme may need caching: you don't want to eval js everytime a user requests for a theme template
// locales may need caching: you don't want to query the locales tables literally every request. locales barely change.

// experiment with going full hypergo for HTML
