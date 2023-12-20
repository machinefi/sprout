pragma solidity >=0.8.0;

import "./interfaces/IStrategyManager.sol";
import "./interfaces/IStrategy.sol";

abstract contract StrategyManagerStorage is IStrategyManager {
    bytes32 public constant DOMAIN_TYPEHASH =
        keccak256("EIP712Domain(string name,uint256 chainId,address verifyingContract)");
    bytes32 public constant DEPOSIT_TYPEHASH =
        keccak256("Deposit(address strategy,address token,uint256 amount,uint256 nonce,uint256 expiry)");
    uint8 internal constant MAX_STAKER_STRATEGY_LIST_LENGTH = 32;

    bytes32 internal _DOMAIN_SEPARATOR;
    mapping(address => uint256) public nonces;
    address public strategyWhitelister;
    mapping(address => mapping(IStrategy => uint256)) public stakerStrategyShares;
    mapping(address => IStrategy[]) public stakerStrategyList;
    mapping(IStrategy => bool) public strategyIsWhitelistedForDeposit;

    constructor(ISlasher _slasher) {
        slasher = _slasher;
    }

    uint256[40] private __gap;
}