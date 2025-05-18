# Prometheus TwCLI Exporter

Exposes metrics for 3Ware 9000 series RAID controllers.
You will require the tw-cli utility installed for the exporter to obtain the various metrics.

## Quick Start

- Build the package
    ```
    make
    ```

- Run the exporter
    ```
    ./prometheus-twcli-exporter
    ```

- Access metrics on `http://localhost:9400/metrics`

## Metrics

| Name                                     | Description                                                    |
|------------------------------------------|----------------------------------------------------------------|
| tw_cli_scrape_collector_success          | Indicates whether the last scrape was successful               |
| tw_cli_scrape_collector_duration_seconds | Time taken to perform last scrape                              |
| tw_cli_controller_info                   | General information regarding controller                       |
| tw_cli_unit_percent_complete             | If unit is REBUILDING/ VERIFYING return percent complete value |
| tw_cli_unit_status                       | Indicates unit health                                          |
| tw_cli_drive_status                      | Indicates physical status                                      |
| tw_cli_drive_power_on_hours              | Power on hours data via SMART data from controller             |
| tw_cli_drive_reallocated_sectors         | Reallocated sector data via SMART data from controller         |
| tw_cli_drive_temperature                 | Drive temperature data via SMART data from controller          |

## Compatibility

The exporter has been verified to work with the following models:

- 9650SE-4LPML

Exporter currently only supports SMART data for SATA drives.
