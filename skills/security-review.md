Run a security-focused review of the code at $ARGUMENTS (file path, directory, or diff).

Work through each category. Report findings with file, line number, risk level (critical / high / medium / low), and a concrete remediation step.

**Injection**
- SQL / NoSQL injection: are all queries parameterised?
- Command injection: is any user input passed to shell commands?
- XSS: is user-controlled data rendered as HTML without escaping?
- Template injection: is user input rendered in a template engine?

**Authentication and authorisation**
- Are protected routes / functions gated by auth checks?
- Are there any insecure direct object references (can a user access another user's data by changing an ID)?
- Are JWTs validated (signature, expiry, algorithm)?
- Are passwords hashed with a strong algorithm (bcrypt, argon2)?

**Sensitive data**
- Are secrets, API keys, or credentials hardcoded?
- Is sensitive data (PII, tokens) logged?
- Is sensitive data included in URLs or error messages?

**Dependencies**
- Are any imports from packages known to have vulnerabilities (flag if you recognise any)?

**Configuration**
- Are security headers set (CSP, HSTS, X-Frame-Options)?
- Is CORS configured restrictively?

End with a risk summary: list critical and high findings first. If none found, state that explicitly.
