# alnvdl/varys

Varys is a barebones RSS reader written in Go. It has (pratically) no external
dependencies and provides an equally barebones web experience. It is meant to
be self-hosted and used by a single user.

## Deploying in Azure App Service

1. Deploy the app in Azure following the [quick start guide](https://learn.microsoft.com/en-us/azure/app-service/quickstart-custom-container?tabs=dotnet&pivots=container-linux-azure-portal).
   When selecting the container, input `ghcr.io` as the registry and `alnvdl/varys:main` as the image, leaving the startup command blank.

2. Make sure to set the following environment variables in the deployment:
   | Environment variable                  | Value
   | -                                     | -
   | `ACCESS_TOKEN`                        | A random secret value.
   | `SESSION_KEY`                         | Another random secret value
   | `DB_PATH`                             | `/home/db.json`
   | `FEEDS`                               | The JSON content of your feedlist.
   | `PORT`                                | `80`
   | `PERSIST_INTERVAL`                    | `15m`
   | `REFRESH_INTERVAL`                    | `20m`
   | `WEBSITES_ENABLE_APP_SERVICE_STORAGE` | `true`

   To generate secret random values, you can run `openssl rand 32 | base64`.

3. While not being required, you may want to enable log persistence as well by following this [guide](https://learn.microsoft.com/en-us/azure/app-service/troubleshoot-diagnostic-logs#enable-application-logging-linuxcontainer).

4. You may need to restart the application to make sure it works well.
