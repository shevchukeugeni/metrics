package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"

	"go.uber.org/zap"

	"github.com/shevchukeugeni/metrics/internal/store"
	"github.com/shevchukeugeni/metrics/internal/types"
)

type DBStore struct {
	logger *zap.Logger
	db     *sql.DB
}

func NewStore(logger *zap.Logger, db *sql.DB) *DBStore {
	return &DBStore{
		logger: logger,
		db:     db,
	}
}

func (dbs *DBStore) GetMetric(mtype string) map[string]string {
	metrics := make(map[string]string)

	rows, err := dbs.db.Query("SELECT name, value from metrics WHERE type=$1", mtype)
	if err != nil {
		dbs.logger.Error("failed to select from database", zap.Error(err))
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		err = rows.Scan(&name, &value)
		if err != nil {
			dbs.logger.Error("failed to scan", zap.Error(err))
			return nil
		}

		metrics[name] = fmt.Sprint(value)
	}

	if err := rows.Err(); err != nil {
		dbs.logger.Error("rowserrorcheck", zap.Error(err))
		return nil
	}

	return metrics
}

func (dbs *DBStore) GetMetrics() map[string]store.Metric {
	gauge := make(map[string]float64)
	counter := make(map[string]int64)

	rows, err := dbs.db.Query("SELECT * from metrics")
	if err != nil {
		dbs.logger.Error("failed to select from database", zap.Error(err))
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var mtype, name string
		var value float64
		err = rows.Scan(&mtype, &name, &value)
		if err != nil {
			dbs.logger.Error("failed to scan", zap.Error(err))
			return nil
		}

		switch mtype {
		case types.Gauge:
			gauge[name] = value
		case types.Counter:
			counter[name] = int64(math.Round(value))
		}
	}

	if err := rows.Err(); err != nil {
		dbs.logger.Error("rowserrorcheck", zap.Error(err))
		return nil
	}

	return map[string]store.Metric{
		types.Gauge:   store.Gauge(gauge),
		types.Counter: store.Counter(counter),
	}
}

func (dbs *DBStore) UpdateMetric(mtype, name, value string) (any, error) {
	switch mtype {
	case types.Gauge:
		fValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}

		if name == "" {
			return nil, errors.New("incorrect name")
		}

		_, err = dbs.db.Exec("INSERT INTO metrics (type,name,value) VALUES ($1,$2,$3) "+
			"ON CONFLICT ON CONSTRAINT metric_unique "+
			"DO UPDATE SET value=EXCLUDED.value;", mtype, name, fValue)
		if err != nil {
			return nil, err
		}

		return fValue, nil
	case types.Counter:
		iValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}

		if name == "" {
			return nil, errors.New("incorrect name")
		}

		row := dbs.db.QueryRow("SELECT value FROM metrics WHERE type=$1 and name=$2;", mtype, name)
		var (
			val float64
		)

		err = row.Scan(&val)
		if err != nil {
			if err == sql.ErrNoRows {
				_, err = dbs.db.Exec("INSERT INTO metrics (type,name,value) VALUES ($1,$2,$3);", mtype, name, iValue)
				if err != nil {
					return nil, err
				}
				return iValue, nil
			} else {
				return nil, err
			}
		}

		_, err = dbs.db.Exec("UPDATE metrics SET value=$1 WHERE type=$2 and name =$3;",
			int64(math.Round(val))+iValue, mtype, name)
		if err != nil {
			return nil, err
		}

		return int64(math.Round(val)) + iValue, nil
	default:
		return nil, errors.New("unknown metric type")
	}
}
