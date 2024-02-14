// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";

import {IProjectRegistry} from "./interfaces/IProjectRegistry.sol";

contract ProjectRegistry is IProjectRegistry, ERC721, ReentrancyGuard {
    uint256 private _nextProjectId;

    mapping(uint256 => Project) private projects;

    constructor() ERC721("ProjectToken", "PTK") {
        _nextProjectId = 1;
    }

    event ProjectPaused(uint256 indexed projectId);
    event ProjectUnpaused(uint256 indexed projectId);
    event ProjectUpserted(uint256 indexed projectId, string uri, bytes32 hash);

    modifier onlyProjectOwner(uint256 _projectId) {
        if (!isProjectOwner(msg.sender, _projectId)) {
            revert OnlyOwnerAllowed();
        }
        _;
    }

    modifier onlyNotEmptyUri(string memory _uri) {
        if (bytes(_uri).length == 0) {
            revert EmptyUriValue();
        }
        _;
    }

    function isProjectOwner(address _account, uint256 _projectId) public view override returns (bool) {
        return ownerOf(_projectId) == _account;
    }

    function createProject(string memory _uri, bytes32 _hash) public nonReentrant onlyNotEmptyUri(_uri) {
        uint256 projectId = _nextProjectId++;
        Project storage newProject = projects[projectId];
        newProject.uri = _uri;
        newProject.hash = _hash;

        _mint(msg.sender, projectId);
        emit ProjectUpserted(projectId, _uri, _hash);
    }

    function pauseProject(uint256 _projectId) public onlyProjectOwner(_projectId) {
        Project storage project = projects[_projectId];

        if (project.paused) {
            revert ProjectAlreadyPaused();
        }

        project.paused = true;
        emit ProjectPaused(_projectId);
    }

    function unpauseProject(uint256 _projectId) public onlyProjectOwner(_projectId) {
        Project storage project = projects[_projectId];

        if (!project.paused) {
            revert ProjectNotPaused();
        }

        project.paused = false;
        emit ProjectUnpaused(_projectId);
    }

    function updateProject(
        uint256 _projectId,
        string memory _uri,
        bytes32 _hash
    ) public onlyProjectOwner(_projectId) onlyNotEmptyUri(_uri) {
        projects[_projectId].uri = _uri;
        projects[_projectId].hash = _hash;
        emit ProjectUpserted(_projectId, _uri, _hash);
    }

    function getProject(uint256 _projectId) public view returns (Project memory) {
        return projects[_projectId];
    }
}