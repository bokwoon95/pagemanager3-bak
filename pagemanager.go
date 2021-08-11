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
// locales caching should be an implementation detail. Don't cache it directly in the application! By keeping the caching behind an interface it opens the possibility of the cache being in redis or soemthing.

// experiment with going full hypergo for HTML
