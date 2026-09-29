# Staff password update bug

Observed in Manage Staff while editing an existing user:

- Leaving **New Password** empty displayed “Password must be at least 6 characters,” despite the form saying blank keeps the existing password.
- Entering a valid replacement password submitted the form but returned HTTP 404.

Expected behavior:

- A blank or whitespace-only edit value must omit the password field and preserve the existing password hash.
- A replacement password must contain at least six characters.
- Updating a user must call the backend's registered HTTP method and route.

Follow-up login report:

- After attempting the password edit, the owner could not sign in using either attempted password.
- Login failure feedback should clearly say “Username or password is incorrect” without revealing which account names exist.
- The owner wants the browser session retained so reopening the POS does not require signing in each time.
