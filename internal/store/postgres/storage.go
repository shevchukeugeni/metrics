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
	tx, err := dbs.db.Begin()
	if err != nil {
		return nil, err
	}
	val, err := updateMetric(tx, mtype, name, value)
	if err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			dbs.logger.Error("tx rollback err", zap.Error(err2))
		}
		return nil, err
	}
	if err2 := tx.Commit(); err2 != nil {
		dbs.logger.Error("tx commit err", zap.Error(err2))
		return nil, err2
	}
	return val, nil
}

func (dbs *DBStore) UpdateMetrics(metrics []types.Metrics) error {
	tx, err := dbs.db.Begin()
	if err != nil {
		return err
	}

	for _, mtr := range metrics {
		var val string
		switch mtr.MType {
		case types.Gauge:
			val = fmt.Sprint(*mtr.Value)
		case types.Counter:
			val = fmt.Sprint(*mtr.Delta)
		default:
			return types.ErrUnknownType
		}
		_, err = updateMetric(tx, mtr.MType, mtr.ID, val)
		if err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				dbs.logger.Error("tx rollback err", zap.Error(err2))
			}
			return err
		}
	}

	if err2 := tx.Commit(); err2 != nil {
		dbs.logger.Error("tx commit err", zap.Error(err2))
		return err2
	}

	return nil
}

func updateMetric(tx *sql.Tx, mtype, name, value string) (any, error) {
	switch mtype {
	case types.Gauge:
		fValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}

		if name == "" {
			return nil, errors.New("incorrect name")
		}

		_, err = tx.Exec("INSERT INTO metrics (type,name,value) VALUES ($1,$2,$3) "+
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

		row := tx.QueryRow("SELECT value FROM metrics WHERE type=$1 and name=$2;", mtype, name)
		var (
			val float64
		)

		err = row.Scan(&val)
		if err != nil {
			if err == sql.ErrNoRows {
				_, err = tx.Exec("INSERT INTO metrics (type,name,value) VALUES ($1,$2,$3);", mtype, name, iValue)
				if err != nil {
					return nil, err
				}
				return iValue, nil
			} else {
				return nil, err
			}
		}

		_, err = tx.Exec("UPDATE metrics SET value=$1 WHERE type=$2 and name =$3;",
			int64(math.Round(val))+iValue, mtype, name)
		if err != nil {
			return nil, err
		}

		return int64(math.Round(val)) + iValue, nil
	default:
		return nil, types.ErrUnknownType
	}
}
