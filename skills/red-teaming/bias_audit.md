---
name: bias_audit
description: "AI bias auditing and fairness"
version: "1.0"
trigger: "bias audit fairness ai model"
platforms: []
requires_tools: ["run_command"]
---

# AI bias auditing and fairness

## Purpose
AI bias auditing and fairness for identifying vulnerabilities and ensuring AI system safety.

## Instructions
1. Define the assessment scope and criteria
2. Design test cases covering known attack vectors
3. Execute tests systematically
4. Document findings with severity ratings
5. Recommend mitigations and verify effectiveness

## Assessment Areas
- Input handling and validation
- Output safety and filtering
- Access control and authorization
- Data privacy and leakage
- Robustness under adversarial conditions

## Testing Methods
- Automated scanning and fuzzing
- Manual testing with crafted inputs
- Statistical analysis of outputs
- Comparison against baselines
- Red team exercises with domain experts

## Reporting
- Severity classification (Critical, High, Medium, Low)
- Detailed reproduction steps
- Impact assessment
- Recommended mitigations
- Verification of fix effectiveness

## Best Practices
- Test systematically, not randomly
- Document all findings thoroughly
- Prioritize by severity and exploitability
- Retest after mitigations
- Share findings responsibly
