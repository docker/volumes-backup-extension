# postgres_pgdata_4gb

This directory contains a Docker Compose set-up to generate a Docker volume of 4 GB. It uses Docker compose to spin up a Postgres container which mounts a volume and runs some initialization SQL scripts to generate data.
Once the data has been imported into the Postgres database, we can use the volume which holds that data for internal benchmarking.

The volume has been pushed to DockerHub as an image: [felipecruz/postgres_pgdata_4gb](https://hub.docker.com/repository/docker/felipecruz/postgres_pgdata_4gb)

See `BenchmarkExportVolume` in [export_test.go](../../../vm/internal/handler/export_test.go).