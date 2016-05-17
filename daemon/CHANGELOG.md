2015/07/28 : 1.0

- FEATURE: enable `~/apps/.provisioning_directives.json` to override real feature data and be updated in real time without provisioning a new stack
- FEATURE: get instance identity API `GET /config/identity`
- FEATURE: get feature sets API `GET /config/features`
- FEATURE: get features in set API `GET /config/features/:feature_set`
- FEATURE: get features in set for user API `GET /config/features/:feature_set/:user_id`
- FEATURE: set key API `PUT /keys/*key`
- FEATURE: get key API `GET /keys/*key`
- FEATURE: watch key API `GET /watchkeys/*key`
- PERFORMANCE: feature data caching so clients don't have to
