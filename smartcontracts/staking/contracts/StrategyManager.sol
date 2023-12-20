pragma solidity >=0.8.0;

import "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/Pausable.sol";
import "./StrategyManagerStorage.sol";
import "./utils/EIP1271SignatureUtils.sol";

contract StrategyManager is
    Initializable,
    Ownable,
    ReentrancyGuard,
    Pausable,
    StrategyManagerStorage
{
    using SafeERC20 for IERC20;

    uint256 internal immutable ORIGINAL_CHAIN_ID;

    modifier onlyStrategyWhitelister() {
        require(
            msg.sender == strategyWhitelister,
            "not the whitelister"
        );
        _;
    }

    modifier onlyStrategiesWhitelistedForDeposit(IStrategy strategy) {
        require(
            strategyIsWhitelistedForDeposit[strategy],
            "strategy not whitelisted"
        );
        _;
    }


    constructor(
    ) StrategyManagerStorage() Ownable(msg.sender) {
        _disableInitializers();
        ORIGINAL_CHAIN_ID = block.chainid;
    }

   function initialize(
        address initialOwner,
        address initialStrategyWhitelister
    ) external initializer {
        _DOMAIN_SEPARATOR = _calculateDomainSeparator();
        _transferOwnership(initialOwner);
        _setStrategyWhitelister(initialStrategyWhitelister);
    }

    function depositIntoStrategy(
        IStrategy strategy,
        IERC20 token,
        uint256 amount
    ) external whenNotPaused nonReentrant returns (uint256 shares) {
        shares = _depositIntoStrategy(msg.sender, strategy, token, amount);
    }

    function depositIntoStrategyWithSignature(
        IStrategy strategy,
        IERC20 token,
        uint256 amount,
        address staker,
        uint256 expiry,
        bytes memory signature
    ) external whenNotPaused nonReentrant returns (uint256 shares) {
        require(expiry >= block.timestamp, "StrategyManager.depositIntoStrategyWithSignature: signature expired");
        uint256 nonce = nonces[staker];
        bytes32 structHash = keccak256(abi.encode(DEPOSIT_TYPEHASH, strategy, token, amount, nonce, expiry));
        unchecked {
            nonces[staker] = nonce + 1;
        }

        bytes32 digestHash = keccak256(abi.encodePacked("\x19\x01", domainSeparator(), structHash));

        /**
         * check validity of signature:
         * 1) if `staker` is an EOA, then `signature` must be a valid ECDSA signature from `staker`,
         * indicating their intention for this action
         * 2) if `staker` is a contract, then `signature` will be checked according to EIP-1271
         */
        EIP1271SignatureUtils.checkSignature_EIP1271(staker, digestHash, signature);

        // deposit the tokens (from the `msg.sender`) and credit the new shares to the `staker`
        shares = _depositIntoStrategy(staker, strategy, token, amount);
    }

    /**
     * @notice Owner-only function to change the `strategyWhitelister` address.
     * @param newStrategyWhitelister new address for the `strategyWhitelister`.
     */
    function setStrategyWhitelister(address newStrategyWhitelister) external onlyOwner {
        _setStrategyWhitelister(newStrategyWhitelister);
    }

    /**
     * @notice Owner-only function that adds the provided Strategies to the 'whitelist' of strategies that stakers can deposit into
     * @param strategiesToWhitelist Strategies that will be added to the `strategyIsWhitelistedForDeposit` mapping (if they aren't in it already)
     */
    function addStrategiesToDepositWhitelist(
        IStrategy[] calldata strategiesToWhitelist
    ) external onlyStrategyWhitelister {
        uint256 strategiesToWhitelistLength = strategiesToWhitelist.length;
        for (uint256 i = 0; i < strategiesToWhitelistLength; ) {
            // change storage and emit event only if strategy is not already in whitelist
            if (!strategyIsWhitelistedForDeposit[strategiesToWhitelist[i]]) {
                strategyIsWhitelistedForDeposit[strategiesToWhitelist[i]] = true;
                emit StrategyAddedToDepositWhitelist(strategiesToWhitelist[i]);
            }
            unchecked {
                ++i;
            }
        }
    }

    /**
     * @notice Owner-only function that removes the provided Strategies from the 'whitelist' of strategies that stakers can deposit into
     * @param strategiesToRemoveFromWhitelist Strategies that will be removed to the `strategyIsWhitelistedForDeposit` mapping (if they are in it)
     */
    function removeStrategiesFromDepositWhitelist(
        IStrategy[] calldata strategiesToRemoveFromWhitelist
    ) external onlyStrategyWhitelister {
        uint256 strategiesToRemoveFromWhitelistLength = strategiesToRemoveFromWhitelist.length;
        for (uint256 i = 0; i < strategiesToRemoveFromWhitelistLength; ) {
            // change storage and emit event only if strategy is already in whitelist
            if (strategyIsWhitelistedForDeposit[strategiesToRemoveFromWhitelist[i]]) {
                strategyIsWhitelistedForDeposit[strategiesToRemoveFromWhitelist[i]] = false;
                emit StrategyRemovedFromDepositWhitelist(strategiesToRemoveFromWhitelist[i]);
            }
            unchecked {
                ++i;
            }
        }
    }

    // INTERNAL FUNCTIONS

    /**
     * @notice This function adds `shares` for a given `strategy` to the `staker` and runs through the necessary update logic.
     * @param staker The address to add shares to
     * @param strategy The Strategy in which the `staker` is receiving shares
     * @param shares The amount of shares to grant to the `staker`
     * @dev In particular, this function calls `delegation.increaseDelegatedShares(staker, strategy, shares)` to ensure that all
     * delegated shares are tracked, increases the stored share amount in `stakerStrategyShares[staker][strategy]`, and adds `strategy`
     * to the `staker`'s list of strategies, if it is not in the list already.
     */
    function _addShares(address staker, IStrategy strategy, uint256 shares) internal {
        require(staker != address(0), "StrategyManager._addShares: staker cannot be zero address");
        require(shares != 0, "StrategyManager._addShares: shares should not be zero!");

        if (stakerStrategyShares[staker][strategy] == 0) {
            require(
                stakerStrategyList[staker].length < MAX_STAKER_STRATEGY_LIST_LENGTH,
                "StrategyManager._addShares: deposit would exceed MAX_STAKER_STRATEGY_LIST_LENGTH"
            );
            stakerStrategyList[staker].push(strategy);
        }

        stakerStrategyShares[staker][strategy] += shares;
    }

    /**
     * @notice Internal function in which `amount` of ERC20 `token` is transferred from `msg.sender` to the Strategy-type contract
     * `strategy`, with the resulting shares credited to `staker`.
     * @param staker The address that will be credited with the new shares.
     * @param strategy The Strategy contract to deposit into.
     * @param token The ERC20 token to deposit.
     * @param amount The amount of `token` to deposit.
     * @return shares The amount of *new* shares in `strategy` that have been credited to the `staker`.
     */
    function _depositIntoStrategy(
        address staker,
        IStrategy strategy,
        IERC20 token,
        uint256 amount
    ) internal onlyStrategiesWhitelistedForDeposit(strategy) returns (uint256 shares) {
        // transfer tokens from the sender to the strategy
        token.safeTransferFrom(msg.sender, address(strategy), amount);

        // deposit the assets into the specified strategy and get the equivalent amount of shares in that strategy
        shares = strategy.deposit(token, amount);

        // add the returned shares to the staker's existing shares for this strategy
        _addShares(staker, strategy, shares);

        // Increase shares delegated to operator, if needed
        // delegation.increaseDelegatedShares(staker, strategy, shares);

        emit Deposit(staker, token, strategy, shares);
        return shares;
    }

    /**
     * @notice Decreases the shares that `staker` holds in `strategy` by `shareAmount`.
     * @param staker The address to decrement shares from
     * @param strategy The strategy for which the `staker`'s shares are being decremented
     * @param shareAmount The amount of shares to decrement
     * @dev If the amount of shares represents all of the staker`s shares in said strategy,
     * then the strategy is removed from stakerStrategyList[staker] and 'true' is returned. Otherwise 'false' is returned.
     */
    function _removeShares(
        address staker,
        IStrategy strategy,
        uint256 shareAmount
    ) internal returns (bool) {
        // sanity checks on inputs
        require(shareAmount != 0, "StrategyManager._removeShares: shareAmount should not be zero!");

        //check that the user has sufficient shares
        uint256 userShares = stakerStrategyShares[staker][strategy];

        require(shareAmount <= userShares, "StrategyManager._removeShares: shareAmount too high");
        //unchecked arithmetic since we just checked this above
        unchecked {
            userShares = userShares - shareAmount;
        }

        // subtract the shares from the staker's existing shares for this strategy
        stakerStrategyShares[staker][strategy] = userShares;

        // if no existing shares, remove the strategy from the staker's dynamic array of strategies
        if (userShares == 0) {
            _removeStrategyFromStakerStrategyList(staker, strategy);

            // return true in the event that the strategy was removed from stakerStrategyList[staker]
            return true;
        }
        // return false in the event that the strategy was *not* removed from stakerStrategyList[staker]
        return false;
    }

    /**
     * @notice Removes `strategy` from `staker`'s dynamic array of strategies, i.e. from `stakerStrategyList[staker]`
     * @param staker The user whose array will have an entry removed
     * @param strategy The Strategy to remove from `stakerStrategyList[staker]`
     */
    function _removeStrategyFromStakerStrategyList(
        address staker,
        IStrategy strategy
    ) internal {
        //loop through all of the strategies, find the right one, then replace
        uint256 stratsLength = stakerStrategyList[staker].length;
        uint256 j = 0;
        for (; j < stratsLength; ) {
            if (stakerStrategyList[staker][j] == strategy) {
                //replace the strategy with the last strategy in the list
                stakerStrategyList[staker][j] = stakerStrategyList[staker][
                    stakerStrategyList[staker].length - 1
                ];
                break;
            }
            unchecked { ++j; }
        }
        // if we didn't find the strategy, revert
        require(j != stratsLength, "StrategyManager._removeStrategyFromStakerStrategyList: strategy not found");
        // pop off the last entry in the list of strategies
        stakerStrategyList[staker].pop();
    }

    /**
     * @notice Internal function for modifying the `strategyWhitelister`. Used inside of the `setStrategyWhitelister` and `initialize` functions.
     * @param newStrategyWhitelister The new address for the `strategyWhitelister` to take.
     */
    function _setStrategyWhitelister(address newStrategyWhitelister) internal {
        emit StrategyWhitelisterChanged(strategyWhitelister, newStrategyWhitelister);
        strategyWhitelister = newStrategyWhitelister;
    }

    // VIEW FUNCTIONS
    /**
     * @notice Get all details on the staker's deposits and corresponding shares
     * @param staker The staker of interest, whose deposits this function will fetch
     * @return (staker's strategies, shares in these strategies)
     */
    function getDeposits(address staker) external view returns (IStrategy[] memory, uint256[] memory) {
        uint256 strategiesLength = stakerStrategyList[staker].length;
        uint256[] memory shares = new uint256[](strategiesLength);

        for (uint256 i = 0; i < strategiesLength; ) {
            shares[i] = stakerStrategyShares[staker][stakerStrategyList[staker][i]];
            unchecked {
                ++i;
            }
        }
        return (stakerStrategyList[staker], shares);
    }

    /// @notice Simple getter function that returns `stakerStrategyList[staker].length`.
    function stakerStrategyListLength(address staker) external view returns (uint256) {
        return stakerStrategyList[staker].length;
    }

    /**
     * @notice Getter function for the current EIP-712 domain separator for this contract.
     * @dev The domain separator will change in the event of a fork that changes the ChainID.
     */
    function domainSeparator() public view returns (bytes32) {
        if (block.chainid == ORIGINAL_CHAIN_ID) {
            return _DOMAIN_SEPARATOR;
        } else {
            return _calculateDomainSeparator();
        }
    }

    // @notice Internal function for calculating the current domain separator of this contract
    function _calculateDomainSeparator() internal view returns (bytes32) {
        return keccak256(abi.encode(DOMAIN_TYPEHASH, keccak256(bytes("EigenLayer")), block.chainid, address(this)));
    }

}