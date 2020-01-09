# Nginx log collector

make help

In table_schema.sql _SET_ME_ change to '{replica}'

## Work scheme
![schema.jpg](doc/schema.jpg?v3)


"logs_cluster" (from table_schema.sql) get from clickhouse_remote_servers.xml between "remote_servers" and "shard"
