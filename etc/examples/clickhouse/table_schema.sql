CREATE DATABASE IF NOT EXISTS nginx;

set allow_experimental_low_cardinality_type=1;

CREATE TABLE nginx.access_log_shard
(
    event_datetime DateTime,
    event_date Date,
    server_name LowCardinality(String),
    remote_user String,
    http_x_real_ip UInt32,
    remote_addr UInt32,
    status UInt16,
    scheme LowCardinality(String),
    request_method LowCardinality(String),
    request_uri String,
    request_args String,
    server_protocol LowCardinality(String),
    body_bytes_sent UInt64,
    request_bytes UInt64,
    http_referer String,
    http_user_agent LowCardinality(String),
    request_time Float32,
    upstream_response_time Array(Float32),
    hostname LowCardinality(String),
    host LowCardinality(String),
    upstream_addr LowCardinality(String)
)
ENGINE = MergeTree(event_date, (hostname, request_uri, event_date), 8192)


CREATE TABLE nginx.access_log
(
    event_datetime DateTime,
    event_date Date,
    server_name LowCardinality(String),
    remote_user String,
    http_x_real_ip UInt32,
    remote_addr UInt32,
    status UInt16,
    scheme LowCardinality(String),
    request_method LowCardinality(String),
    request_uri String,
    request_args String,
    server_protocol LowCardinality(String),
    body_bytes_sent UInt64,
    request_bytes UInt64,
    http_referer String,
    http_user_agent LowCardinality(String),
    request_time Float32,
    upstream_response_time Array(Float32),
    hostname LowCardinality(String),
    host LowCardinality(String),
    upstream_addr LowCardinality(String)
)
ENGINE = Distributed('logs_cluster', 'nginx', 'access_log_shard', rand())


CREATE TABLE nginx.error_log
(
    event_datetime DateTime,
    event_date Date,
    server_name LowCardinality(String),
    http_referer String,
    pid UInt32,
    sid UInt32,
    tid UInt64,
    host LowCardinality(String),
    client String,
    request String,
    message String,
    login String,
    upstream String,
    subrequest String,
    hostname LowCardinality(String)
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/logs_replicator/nginx.error2_log', '{replica}', event_date, (server_name, request, event_date), 8192)
