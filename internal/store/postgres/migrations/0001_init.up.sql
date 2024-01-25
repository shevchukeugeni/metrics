DO
$$
    BEGIN
        IF NOT EXISTS(SELECT 1 FROM pg_type WHERE typname = 'metric_category') THEN
            CREATE TYPE metric_category AS ENUM ('counter', 'gauge');
        END IF;
    END
$$;

CREATE TABLE IF NOT EXISTS metrics
(
    type metric_category NOT NULL,
    name varchar NOT NULL,
    value double precision,

    CONSTRAINT metric_unique UNIQUE (type,name)
);