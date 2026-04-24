---
name: foundry
description: "Foundry toolkit for Ethereum development"
version: "1.0"
trigger: "foundry forge cast anvil solidity"
platforms: []
requires_tools: ["run_command"]
---

# Foundry Development

## Purpose
Use Foundry (forge, cast, anvil) for fast Solidity development, testing, and deployment.

## Instructions
1. Install Foundry toolchain
2. Initialize project with forge init
3. Write contracts and Solidity tests
4. Use forge test for testing with traces
5. Deploy with forge create or forge script

## Quick Start
```bash
forge init my-project
cd my-project
forge build
forge test -vvv
```

## Testing
```solidity
function testTransfer() public {
    token.mint(address(this), 1000);
    token.transfer(alice, 500);
    assertEq(token.balanceOf(alice), 500);
    assertEq(token.balanceOf(address(this)), 500);
}
```

## Fuzz Testing
```solidity
function testFuzz_Transfer(uint256 amount) public {
    vm.assume(amount <= 1000);
    token.mint(address(this), 1000);
    token.transfer(alice, amount);
    assertEq(token.balanceOf(alice), amount);
}
```

## Best Practices
- Write tests in Solidity for maximum fidelity
- Use forge snapshot for gas benchmarking
- Use anvil for local development and testing
- Use cast for chain interaction and debugging
- Run fuzz tests with high iteration counts before deployment
