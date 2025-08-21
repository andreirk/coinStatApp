-- Initialize ClickHouse tables for coinStatApp

-- Create swaps table with deduplication
CREATE TABLE IF NOT EXISTS swaps (
    id String,
    who String,
    token String,
    amount Float64,
    usd Float64,
    side String,
    timestamp DateTime
) ENGINE = ReplacingMergeTree()
ORDER BY (id, timestamp)
PRIMARY KEY (id);

-- Create materialized view for 5-minute statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS stats_5min
ENGINE = SummingMergeTree()
ORDER BY (token, window_start)
POPULATE AS
SELECT
    token,
    toStartOfInterval(timestamp, INTERVAL 5 MINUTE) as window_start,
    sum(usd) as volume,
    count() as count
FROM swaps
GROUP BY token, window_start;

-- Create materialized view for 1-hour statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS stats_1h
ENGINE = SummingMergeTree()
ORDER BY (token, window_start)
POPULATE AS
SELECT
    token,
    toStartOfInterval(timestamp, INTERVAL 1 HOUR) as window_start,
    sum(usd) as volume,
    count() as count
FROM swaps
GROUP BY token, window_start;

-- Create materialized view for 24-hour statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS stats_24h
ENGINE = SummingMergeTree()
ORDER BY (token, window_start)
POPULATE AS
SELECT
    token,
    toStartOfInterval(timestamp, INTERVAL 1 DAY) as window_start,
    sum(usd) as volume,
    count() as count
FROM swaps
GROUP BY token, window_start;
