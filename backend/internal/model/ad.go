package model

// AdLog represents a row in WEO_AD_LOG table.
type AdLog struct {
	ALSeq    int    `db:"AL_SEQ"`
	MASeq    int    `db:"MA_SEQ"`
	USRSeq   *int   `db:"USR_SEQ"`
	ALType   string `db:"AL_TYPE"`
	ALDate   string `db:"AL_DATE"`
	ALIPAddr string `db:"AL_IPADDR"`
}
