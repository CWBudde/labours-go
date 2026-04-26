# Hercules Compatibility Fixtures

This directory records provenance and expectations for Hercules compatibility fixtures.

At the start of Phase 0, existing sample data lives in:

- `example_data/hercules_burndown.pb`
- `example_data/hercules_burndown.yaml`
- `example_data/hercules_devs.pb`
- `example_data/hercules_devs.yaml`
- `example_data/hercules_couples.pb`
- `data/labours-go_burndown.yaml`
- `data/labours-go_devs.yaml`
- `data/labours-go_couples.yaml`
- `test/testdata/simple_burndown.pb`
- `test/testdata/realistic_burndown.pb`

The current provenance of these files is not fully documented. Before using any file as a compatibility golden, record:

- the Hercules repository path or commit that generated it
- the exact Hercules command
- whether the input repository was local or remote
- the Hercules commit analyzed, when relevant
- whether the fixture is intended for reader tests, chart structure tests, or Python parity tests

Future current-Hercules fixtures should be generated from `../hercules` and kept small enough for normal test runs.

Current checked-in current-Hercules fixtures live in `test/testdata/hercules/`:

- `report_default.pb`: default `hercules report` analysis flag set.
- `shotness.pb`: `--shotness` payload used by report-all modes.

Suggested fixture commands:

```bash
../hercules/hercules --pb --burndown --burndown-files --burndown-people <repo> > testdata/hercules/burndown.pb
../hercules/hercules --pb --couples <repo> > testdata/hercules/couples.pb
../hercules/hercules --pb --devs <repo> > testdata/hercules/devs.pb
../hercules/hercules --pb --temporal-activity --bus-factor --ownership-concentration --knowledge-diffusion --hotspot-risk <repo> > testdata/hercules/report-default-extra.pb
```

See `PLAN.md` for the broader compatibility test strategy.
