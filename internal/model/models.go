package model

import (
	"net"
)

type IPRange struct {
	ID          int64     `db:"id"`
	Network     net.IPNet `db:"network"`
	CountryCode string    `db:"country_code"`
	Version     int       `db:"ip_version"` // 4 or 6
}

type IPResponse struct {
	IP          string `json:"ip"`
	CountryCode string `json:"country_code"`
}

type Error struct {
	Message string `json:"message"`
}
