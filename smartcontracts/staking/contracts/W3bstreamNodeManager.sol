pragma solidity >=0.8.0;

import "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "./interfaces/IStrategyManager.sol";

contract W3bstreamNodeManager is Initializable, ReentrancyGuard {

    IStrategyManager public strategyManager;

    mapping(address => bool) public nodes;

    event NodeRegistered(address indexed node);

    constructor(IStrategyManager _strategyManager) Ownable(msg.sender) {
        _disableInitializers();
        strategyManager = _strategyManager;
    }

    function initialize() external initializer {
    }

    function register() external {
        address node = msg.sender;
        if (!nodes[node]) {
            nodes[node] = true;
            emit NodeRegistered(node);
        }
    }

    function sharesOf(address staker) external returns (uint256) {
        (IStrategy[] memory strategies, uint256[] memory shares) = strategyManager.getDeposits(staker);
        uint256 retval = 0;
        for (uint256 i = 0; i < strategies.length; i++) {
            retval += shares[i];
        }
        return retval;
    }

}
