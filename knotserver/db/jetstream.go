package db

func (d *DB) SaveLastTimeUs(lastTimeUs int64) error {
	_, err := d.db.Exec(`insert into _jetstream (last_time_us) values (?)`, lastTimeUs)
	return err
}

func (d *DB) UpdateLastTimeUs(lastTimeUs int64) error {
	_, err := d.db.Exec(`update _jetstream set last_time_us = ? where rowid = 1`, lastTimeUs)
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) GetLastTimeUs() (int64, error) {
	var lastTimeUs int64
	row := d.db.QueryRow(`select last_time_us from _jetstream`)
	err := row.Scan(&lastTimeUs)
	return lastTimeUs, err
}
