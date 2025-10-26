package examples

func GetLocationQuery() string {

	return `SELECT ST_X(coordinates::geometry) AS longitude, ST_Y(coordinates::geometry) AS latitude FROM users WHERE uuid = $1;`
	// or SELECT ST_X(coordinates::geometry), ST_Y(coordinates::geometry) FROM users WHERE uuid = $1;
}

func InsertLocationQuery() string {

	return `UPDATE users SET coordinates = ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography WHERE uuid = $3;`
	// %1 is Longitude, %2 is Latitude  x,y
}

func FindNearUsers() string {
	/*

	example -----------------------------------------------------------------------------------------------
		type NearbyUser struct {
		Uuid     string
		Email    string
		Distance float64 // meters
	}

	const nearbySQL = `
	SELECT uuid, email,
	       ST_Distance(
	         coordinates,
	         ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
	       ) AS distance_m
	FROM users
	WHERE coordinates IS NOT NULL
	  AND ST_DWithin(
	        coordinates,
	        ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
	        $3  -- radius meters: 300, 500, 1000
	      )
	ORDER BY distance_m
	LIMIT $4;
	`

	func NearbyUsers(db *sql.DB, lon, lat float64, radiusM float64, limit int) ([]NearbyUser, error) {
		rows, err := db.Query(nearbySQL, lon, lat, radiusM, limit)
		if err != nil { return nil, err }
		defer rows.Close()

		var out []NearbyUser
		for rows.Next() {
			var u NearbyUser
			if err := rows.Scan(&u.Uuid, &u.Email, &u.Distance); err != nil {
				return nil, err
			}
			out = append(out, u)
		}
		return out, rows.Err()
	}
	*/
	return `SELECT uuid, email, ST_Distance(coordinates, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance_m FROM users WHERE coordinates IS NOT NULL AND ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3) ORDER BY distance_m LIMIT $4;`
}
