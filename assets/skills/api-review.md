Review the API endpoint(s) at $ARGUMENTS (file path or route) before submitting.

Check each category and report findings with file and line number:

**Input validation**
- Is all input validated at the boundary before use?
- Are validation errors returned with clear, consistent error responses?
- Are path parameters, query strings, and body fields all validated?

**Authentication and authorisation**
- Is the endpoint protected by authentication middleware?
- Is authorisation checked — not just "is the user logged in" but "can this user do this"?
- Are there any privilege escalation risks?

**Error handling**
- Are errors caught and returned as structured responses (not stack traces)?
- Are error messages safe to expose to the client (no internal paths, DB details)?
- Are appropriate HTTP status codes used?

**Data handling**
- Are database queries parameterised?
- Is sensitive data (passwords, tokens) excluded from responses?
- Are responses paginated where the result set could be large?

**Tests**
- Is there a test for the happy path?
- Are auth/permission failure cases tested?
- Are invalid input cases tested?

End with a one-line summary: PASS, PASS WITH SUGGESTIONS, or NEEDS WORK.
