# Google Drive storage setup

Users can select Google Drive without changing vault routes or domain code. The
application uses one designated vault-owner Google account and one app-managed
Drive folder. View and download requests are authenticated by Aether, then
redirected to Drive's own viewer or download URL. Direct Drive edits are not
synchronized back into Aether.

## 1. Create a Google Cloud project

1. Open the [Google Cloud Console](https://console.cloud.google.com/).
2. Create or select a project dedicated to Aether.
3. Enable **Google Drive API** under **APIs & Services → Library**.
4. Configure the OAuth consent screen. For a private family deployment,
   choose **External**, add the vault-owner account as a test user, and keep
   the app in testing until the deployment is ready for production review.
5. Add the scope
   `https://www.googleapis.com/auth/drive.file` when Google requests scopes.

## 2. Create the OAuth client

Under **APIs & Services → Credentials**, create an **OAuth client ID** of type
**Web application**. Add this exact authorized redirect URI:

```text
http://127.0.0.1:8090/auth/gdrive/callback
```

The loopback URI is used only by the one-time credential helper. Do not use a
wildcard redirect, and do not add a query string or fragment.

Save the client ID and client secret in a password manager or deployment
secret store. They are the values for `AETHER_GDRIVE_CLIENT_ID` and
`AETHER_GDRIVE_CLIENT_SECRET`.

## 3. Create the app-managed folder

Sign in to the designated vault-owner Google account at
[drive.google.com](https://drive.google.com/), create a new folder such as
`Aether Vault`, and copy its folder ID from the URL:

```text
https://drive.google.com/drive/folders/<FOLDER_ID>
```

Set that value as `AETHER_GDRIVE_FOLDER_ID`. Share the folder with the Google
accounts that are allowlisted in Aether as viewers so redirected links work for
members. Do not make the folder or its files public.

## 4. Obtain the refresh token

From the repository root, run the short-lived loopback helper:

```bash
cd backend
go run ./cmd/aether-gdrive-auth \
  -client-id "$AETHER_GDRIVE_CLIENT_ID" \
  -client-secret "$AETHER_GDRIVE_CLIENT_SECRET" \
  -redirect-url http://127.0.0.1:8090/auth/gdrive/callback
```

The command prints an authorization URL. Open it in the vault-owner browser,
approve the `drive.file` permission, and wait for the loopback callback. The
command then prints one line:

```text
AETHER_GDRIVE_REFRESH_TOKEN=...
```

Copy the value directly into deployment secret storage. Never commit it,
place it in `.env.example`, send it to the browser, or include it in logs.
If Google does not return a refresh token, revoke the app's prior consent from
the owner's Google account and run the helper again.

## 5. Configure Aether

Set the existing family-login variables and the Drive variables together:

```dotenv
AETHER_STORAGE_PROVIDER=gdrive
AETHER_GDRIVE_CLIENT_ID=your-client-id
AETHER_GDRIVE_CLIENT_SECRET=your-client-secret
AETHER_GDRIVE_REDIRECT_URL=http://127.0.0.1:8090/auth/gdrive/callback
AETHER_GDRIVE_FOLDER_ID=your-folder-id
AETHER_GDRIVE_REFRESH_TOKEN=your-refresh-token
```

The application also requires the normal Google OIDC login settings:
`AETHER_GOOGLE_CLIENT_ID`, `AETHER_GOOGLE_CLIENT_SECRET`,
`AETHER_GOOGLE_REDIRECT_URL`, and `AETHER_BOOTSTRAP_ADMIN_EMAIL`.

Start Aether only after all five Drive variables are present. Configuration
loading rejects partial Drive credentials and rejects redirect URLs with a
different path, query, or fragment.

## Token rotation and revocation

To rotate access, create a new refresh token with the helper, update the
deployment secret, and restart Aether. To revoke access, remove the secret
and revoke Aether under the vault-owner's Google Account security settings.
Existing Drive objects remain in the app-managed folder but are inaccessible
until a valid owner token is configured again.

## Asynchronous trashing

Moving a document to Trash does not wait for Google Drive. The API persists the
soft-deleted catalog state and returns it immediately, then a bounded worker
sets the Drive `trashed` flag for the original and manifest. Pending and failed
work survives process restarts and is retried; restore and permanent purge are
blocked until `deletionStatus` is `complete`.
