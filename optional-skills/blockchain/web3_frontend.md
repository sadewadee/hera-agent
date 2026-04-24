---
name: web3_frontend
description: "Web3 frontend integration"
version: "1.0"
trigger: "web3 frontend wallet connect ethers"
platforms: []
requires_tools: ["run_command"]
---

# Web3 frontend integration

## Purpose
Web3 frontend integration for blockchain and Web3 applications.

## Instructions
1. Define the smart contract or dApp requirements
2. Design the architecture and token economics
3. Implement with security best practices
4. Audit for vulnerabilities before deployment
5. Deploy to testnet first, then mainnet

## Development
- Use established frameworks (Hardhat, Anchor, Foundry)
- Follow secure coding patterns for the target chain
- Write comprehensive tests including edge cases
- Use fuzzing for input validation testing
- Document all external dependencies

## Security
- Check for reentrancy, overflow, and access control issues
- Use well-audited libraries (OpenZeppelin for EVM)
- Implement proper access control and role management
- Handle edge cases in token transfers
- Consider MEV and front-running implications

## Testing
- Unit tests for all contract functions
- Integration tests with simulated blockchain state
- Fuzz testing for input boundaries
- Gas optimization profiling
- Testnet deployment and manual verification

## Deployment
- Deploy to testnet first and verify all functionality
- Use multi-sig for contract ownership
- Implement upgrade patterns (proxy, diamond) if needed
- Document deployment addresses and verification
- Set up monitoring for contract events

## Best Practices
- Start simple and add complexity incrementally
- Get professional security audits before mainnet
- Implement emergency pause functionality
- Use time-locked governance for parameter changes
- Maintain transparent communication with users
