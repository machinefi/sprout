import { expect } from 'chai';
import { loadFixture } from '@nomicfoundation/hardhat-toolbox/network-helpers';
import { ethers } from 'hardhat';
import { SignerWithAddress } from '@nomicfoundation/hardhat-ethers/signers';

import { deployFleetManager, deployW3bstreamRouter } from './deployers';
import { FleetManager, ProjectRegistry, W3bstreamRouter, WSReceiver } from '../typechain-types';
import { registerOperator, registerProject } from './helpers';
import { DIGEST_STRING, JOURNAL_STRING, SEAL_RAW } from './testData';

describe('W3bstreamRouter', function () {
  const PROJECT_1_ID = 1;
  const NODE_ID_1 = 1;

  describe('data submission', function () {
    let router: W3bstreamRouter;
    let projectRegistry: ProjectRegistry;
    let fleet: FleetManager;
    let receiver: WSReceiver;

    beforeEach(async function () {
      const [projectOwner, node, operator] = await ethers.getSigners();

      // 1. deploy operator registry and fleet manager (once)
      fleet = await loadFixture(deployFleetManager);
      // 2. register operator (each new operator) from Profile account
      await registerOperator(await fleet.nodeRegistry(), node, operator);
      // 3. deploy device registry and WSreceiver (each depin project)
      receiver = await loadFixture(deployWSReceiver);
      await registerDevices(receiver, projectOwner);

      // 4. SPROUT upload project config to ipfs (once from sprout node)
      // const { IpfsHash } = await uploadProjectConfig(deviceRegistry, await receiver.getAddress());
      // console.log(IpfsHash);
      const IpfsHash = 'QmcRTH9pmHYihK3ooZRXDSYXMhAX7GSGTF92Ab5Dc4fHPN';
      // 5. register new project, from ProjectOwner account
      const projectRegistryAddr = await fleet.projectRegistry();
      await registerProject(
        projectRegistryAddr,
        projectOwner,
        'ipfs://' + IpfsHash,
        '0x91f11349770aadcc135213916bf429e39f7419b25d5fe6a2623115b35b381389',
      );
      // 6. allow operator in fleet manager, from ProjectOwner account
      await fleet.connect(projectOwner).allow(PROJECT_1_ID, NODE_ID_1);
      // 7. deploy and initialize WSRouter
      router = await loadFixture(deployW3bstreamRouter);
      projectRegistry = await ethers.getContractAt('ProjectRegistry', projectRegistryAddr);
      await router.initialize(await projectRegistry.getAddress(), await fleet.getAddress());
      // 8. Grant role to receiver to mint rewards
      const rewardsAddr = await receiver.tokenAddress();
      const rewards = await ethers.getContractAt('DeviceReward', rewardsAddr);
      const MINTER_ROLE = await rewards.MINTER_ROLE();
      await rewards.grantRole(MINTER_ROLE, receiver);
    });

    it('works', async function () {
      const [, , node] = await ethers.getSigners();

      await router.register(PROJECT_1_ID, await receiver.getAddress());

      const seal = '0x' + Buffer.from(JSON.stringify(SEAL_RAW)).toString('hex');
      const digest = '0x' + Buffer.from(DIGEST_STRING).toString('hex');
      const journal = '0x' + Buffer.from(JOURNAL_STRING).toString('hex');
      const coder = new ethers.AbiCoder();

      await expect(
        router
          .connect(node)
          .submit(
            PROJECT_1_ID,
            await receiver.getAddress(),
            coder.encode(['bytes', 'bytes', 'bytes'], [seal, digest, journal]),
          ),
      )
        .to.emit(router, 'DataReceived')
        .withArgs(node.address, true, '');
    });
  });
});

const deployDeviceRegistry = async () => {
  const DeviceRegistry = await ethers.getContractFactory('DeviceRegistry');
  return DeviceRegistry.deploy();
};

const deployWSReceiver = async () => {
  const deviceRegistry = await deployDeviceRegistry();
  const deviceRewards = await deployERC20();

  const WSReceiver = await ethers.getContractFactory('WSReceiver');
  return WSReceiver.deploy(deviceRegistry, deviceRewards);
};

const deployERC20 = async () => {
  const factory = await ethers.getContractFactory('DeviceReward');
  return factory.deploy();
};

async function registerDevices(receiver: WSReceiver, projectOwner: SignerWithAddress) {
  const deviceRegistryAddr = await receiver.deviceNFTRegistry();
  const deviceRegistry = await ethers.getContractAt('DeviceRegistry', deviceRegistryAddr);

  await deviceRegistry.safeMint('0', projectOwner.address, 'uri');
  await deviceRegistry.safeMint('1', projectOwner.address, 'uri');
  await deviceRegistry.safeMint('2', projectOwner.address, 'uri');
  await deviceRegistry.safeMint('3', projectOwner.address, 'uri');

  await deviceRegistry.enable(0);
  await deviceRegistry.enable(1);
  await deviceRegistry.enable(3);
}
