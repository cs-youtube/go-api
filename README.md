# go-api

A small REST API in Go (standard library), compared under load with
[rust-api](https://github.com/cs-youtube/rust-api) in the
[api-lab](https://github.com/cs-youtube/api-lab) cluster.

| Endpoint | What it does |
| --- | --- |
| `GET /items?limit=N` | The first N of the 1000 items held in memory, as JSON |
| `GET /items/{id}` | One item, or 404 |
| `POST /items` | Parses a JSON item, validates it and returns it |
| `GET /metrics`, `GET /healthz` | Prometheus metrics, health |

A push to `main` runs the tests, builds the image on the lab's self-hosted runner, pushes it to
GHCR and pins its digest in [api-lab-deployment](https://github.com/cs-youtube/api-lab-deployment); Argo CD then deploys it.
Pull requests are tested on GitHub's machines, never on the lab's runner. The endpoints and metric names must
stay the same as rust-api's, or the comparison stops being fair.
