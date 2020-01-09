# Nginx log collector

make help

### Work scheme
![schema.jpg](doc/schema.jpg?v3)

### For ClickHouse server:
"logs_cluster" (from table_schema.sql) get from clickhouse_remote_servers.xml between "remote_servers" and "shard"
