---
name: adversarial
description: "Adversarial examples and robustness"
version: "1.0"
trigger: "adversarial examples attack robustness"
platforms: []
requires_tools: ["run_command"]
---

# Adversarial examples and robustness

## Purpose
Adversarial examples and robustness for identifying and mitigating security vulnerabilities in AI systems.

## Instructions
1. Define the scope and rules of engagement
2. Identify potential attack surfaces and vectors
3. Design and execute test scenarios
4. Document findings with severity ratings
5. Recommend mitigations and defenses

## Attack Taxonomy
- **Input attacks**: Malicious inputs that subvert system behavior
- **Training attacks**: Corruption of training data or process
- **Model attacks**: Extraction, inversion, or manipulation of models
- **System attacks**: Infrastructure and supply chain vulnerabilities
- **Social attacks**: Manipulation of human operators

## Testing Methodology
1. Reconnaissance: understand the target system
2. Threat modeling: identify likely attack vectors
3. Attack development: craft specific test cases
4. Execution: run attacks in controlled environment
5. Analysis: evaluate success rate and impact
6. Reporting: document findings and recommendations

## Defense Strategies
- Input validation and sanitization
- Output filtering and safety classifiers
- Rate limiting and anomaly detection
- Model watermarking and fingerprinting
- Monitoring and incident response

## Ethical Considerations
- Only test systems you have authorization to test
- Follow responsible disclosure practices
- Document all testing activities
- Report vulnerabilities through proper channels
- Do not cause harm to real users or data

## Best Practices
- Use a comprehensive threat model as your testing guide
- Test both known attack patterns and novel approaches
- Include automated and manual testing methods
- Retest after mitigations are implemented
- Maintain an up-to-date attack library
