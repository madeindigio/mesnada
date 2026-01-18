# Mesnada Personas

This directory contains example persona files that define different behaviors and roles for spawned agents.

## What are Personas?

Personas are pre-defined instructions that get prepended to the prompt when spawning an agent. They allow you to give agents specific roles, expertise areas, or behavioral guidelines without having to repeat those instructions in every prompt.

## How to Use Personas

1. Configure the `persona_path` in your mesnada config file:

```yaml
orchestrator:
  persona_path: "~/.mesnada/personas"
```

2. Create `.md` files in the persona directory. The filename (without `.md`) becomes the persona name.

3. When spawning an agent via MCP, specify the persona parameter:

```json
{
  "prompt": "Review the authentication module",
  "persona": "code_reviewer",
  "work_dir": "/path/to/project"
}
```

## Example Personas Included

- **senior_programmer.md** - Experienced engineer focused on best practices, scalability, and maintainability
- **qa_expert.md** - Quality assurance specialist focused on comprehensive testing
- **code_reviewer.md** - Experienced reviewer focused on code quality and constructive feedback

## Creating Your Own Personas

Simply create a new `.md` file in your persona directory with instructions for the agent. For example:

**security_auditor.md**:
```markdown
You are a security auditor specializing in web application security.

Your focus areas:
- OWASP Top 10 vulnerabilities
- Authentication and authorization
- Input validation and sanitization
- Secure communication (TLS, certificates)
- Secret management
- SQL injection and XSS prevention

When reviewing code:
- Look for security vulnerabilities
- Check for proper authentication/authorization
- Verify input validation
- Ensure secrets are not hardcoded
- Check for secure defaults
```

## Best Practices

- Keep personas focused on a specific role or expertise
- Be clear and specific in the instructions
- Include examples of the expected behavior
- Update personas based on experience with them
- Share useful personas with your team
