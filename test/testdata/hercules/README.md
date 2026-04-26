# Hercules-Generated Protobuf Fixtures

These fixtures are generated from the neighboring `../hercules` checkout and are intended for reader/schema compatibility tests.

## `report_default.pb`

Generated on 2026-04-26 from:

```bash
../hercules/hercules \
  --pb --quiet \
  --burndown --burndown-files --burndown-people \
  --couples \
  --devs \
  --temporal-activity \
  --bus-factor \
  --ownership-concentration \
  --knowledge-diffusion \
  --hotspot-risk \
  ../hercules/cmd/hercules/test_data/hercules.siva \
  > test/testdata/hercules/report_default.pb
```

The source repository fixture is `../hercules/cmd/hercules/test_data/hercules.siva`.
This file represents the default analysis flag set used by `../hercules/cmd/hercules/report.go`.

## `shotness.pb`

Generated on 2026-04-26 from:

```bash
../hercules/hercules \
  --pb --quiet \
  --shotness \
  ../hercules/cmd/hercules/test_data/hercules.siva \
  > test/testdata/hercules/shotness.pb
```

The source repository fixture is `../hercules/cmd/hercules/test_data/hercules.siva`.
This file covers the `ShotnessAnalysisResults` payload used by `shotness` and `couples-shotness` in report-all workflows.
