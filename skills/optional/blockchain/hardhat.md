---
name: hardhat
description: "Hardhat Ethereum development environment"
version: "1.0"
trigger: "hardhat ethereum development testing"
platforms: []
requires_tools: ["run_command"]
---

# Hardhat Development

## Purpose
Use Hardhat for Ethereum smart contract development, testing, and deployment.

## Instructions
1. Initialize Hardhat project
2. Write and compile smart contracts
3. Create comprehensive tests
4. Deploy to test and main networks
5. Verify contracts on block explorers

## Setup
```bash
npx hardhat init
```

## Testing
```javascript
describe("Token", function () {
  it("should deploy with correct supply", async function () {
    const Token = await ethers.getContractFactory("Token");
    const token = await Token.deploy(1000);
    expect(await token.totalSupply()).to.equal(1000);
  });
});
```

## Best Practices
- Write tests for all contract functions
- Use hardhat-gas-reporter for gas optimization
- Deploy to testnets before mainnet
- Use environment variables for private keys
- Verify contracts on Etherscan after deployment
