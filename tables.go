package pagemanager

import "github.com/bokwoon95/pagemanager/sq"

type pm_SUPERADMIN struct {
	sq.TableInfo
	ORDER_NUM     sq.NumberField `sq:"type=INTEGER misc=PRIMARY_KEY"`
	LOGIN_ID      sq.StringField
	PASSWORD_HASH sq.StringField
	KEY_PARAMS    sq.StringField
}

func new_SUPERADMIN(alias string) pm_SUPERADMIN {
	tbl := pm_SUPERADMIN{TableInfo: sq.TableInfo{Alias: alias}}
	tbl.TableInfo.Name = "pm_superadmin"
	_ = sq.ReflectTable(&tbl)
	return tbl
}

type pm_KEYS struct {
	sq.TableInfo
	KEY_ID         sq.StringField `sq:"type=TEXT misc=PRIMARY_KEY"`
	KEY_CIPHERTEXT sq.StringField
	STATUS         sq.NumberField
	CREATED_AT     sq.TimeField
}

func new_KEYS(alias string) pm_KEYS {
	tbl := pm_KEYS{TableInfo: sq.TableInfo{Alias: alias}}
	tbl.TableInfo.Name = "pm_keys"
	_ = sq.ReflectTable(&tbl)
	return tbl
}
