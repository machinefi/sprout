// SPDX-License-Identifier: BUSL-1.1
pragma solidity >=0.8.0;

import "../interfaces/IStrategy.sol";
import "../interfaces/IStrategyManager.sol";
import "@openzeppelin/contracts/utils/Pausable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/proxy/utils/Initializable.sol";

contract CoinStrategy is Initializable, Pausable, IStrategy {
    using SafeERC20 for IERC20;

    uint256 internal constant SHARES_OFFSET = 1e3;
    uint256 internal constant BALANCE_OFFSET = 1e3;
    IStrategyManager public immutable strategyManager;
    uint256 public totalShares;
    uint256[48] private __gap;

    modifier onlyStrategyManager() {
        require(msg.sender == address(strategyManager), "only strategy manager");
        _;
    }

    constructor(IStrategyManager _strategyManager) {
        strategyManager = _strategyManager;
    }

    function initialize() public initializer {
    }

    function deposit(
        IERC20 token,
        uint256 amount
    ) external override whenNotPaused onlyStrategyManager returns (uint256 newShares) {
        require(address(token) == address(0), "deposit coin only");

        newShares = (amount * (totalShares + SHARES_OFFSET)) / (_coinBalance() + BALANCE_OFFSET - amount);
        require(newShares != 0, "new shares cannot be zero");
        totalShares += newShares;
        return newShares;
    }

    function withdraw(
        address recipient,
        IERC20 token,
        uint256 amountShares
    ) external virtual override whenNotPaused onlyStrategyManager {
        require(address(token) == address(0), "StrategyBase.withdraw: Can only withdraw the strategy token");
        uint256 total = totalShares;
        require(amountShares <= total, "invalid shares");
        uint256 amountToSend = (amountShares * (_coinBalance() + BALANCE_OFFSET)) / (total + SHARES_OFFSET);
        totalShares = total - amountShares;
        (bool sent, ) = recipient.call{value: amountToSend}("");
        require(sent, "Failed to send Ether");
    }

    function explanation() external pure virtual override returns (string memory) {
        return "Coin Strategy implementation";
    }

    function sharesToUnderlyingView(uint256 amountShares) public view virtual override returns (uint256) {
        return ((totalShares + SHARES_OFFSET) * amountShares) / (_coinBalance() + BALANCE_OFFSET);
    }

    function sharesToUnderlying(uint256 amountShares) public view virtual override returns (uint256) {
        return sharesToUnderlyingView(amountShares);
    }

    function underlyingToken() external pure returns (IERC20) {
        return IERC20(address(0));
    }

    function underlyingToSharesView(uint256 amountUnderlying) public view virtual returns (uint256) {
        return (amountUnderlying * (totalShares + SHARES_OFFSET)) / (_coinBalance() + BALANCE_OFFSET);
    }

    function underlyingToShares(uint256 amountUnderlying) external view virtual returns (uint256) {
        return underlyingToSharesView(amountUnderlying);
    }

    function userUnderlyingView(address user) external view virtual returns (uint256) {
        return sharesToUnderlyingView(shares(user));
    }

    function userUnderlying(address user) external virtual returns (uint256) {
        return sharesToUnderlying(shares(user));
    }

    function shares(address user) public view virtual returns (uint256) {
        return strategyManager.stakerStrategyShares(user, IStrategy(address(this)));
    }

    function _coinBalance() internal view virtual returns (uint256) {
        return address(this).balance;
    }
}