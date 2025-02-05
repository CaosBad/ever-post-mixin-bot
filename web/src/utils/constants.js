// model alike
const I18N_OPT = [{
    label: '中文简体',
    value: 'zh'
  },
  {
    label: 'English',
    value: 'en'
  }
]

const mixinHost = 'https://api.mixin.one'
const serverHost = '/api'
// const serverHost = 'http://localhost:7008'
const IPFS_NODE = 'ipfs.infura.io'
const IPFS_GATEWAY = 'https://ipfs.io/ipfs/'
// const dappId = {
//   development: '91a9aeeaa6609ebd73afc7c542eec02e7510c23a65d2e1db2840c46f74f67ceb',
//   production: '91a9aeeaa6609ebd73afc7c542eec02e7510c23a65d2e1db2840c46f74f67ceb'
// }

const BOT = {
  clientId: 'ee7ef479-e9e7-44ba-9755-6be4e1d826d7',
  scope: 'PROFILE:READ+ASSETS:READ'
}

const ASSETS = {
  CNB: '965e5c6e-434c-3fa9-b780-c50f43cd955c',
  XIN: 'c94ac88f-4671-3976-b60a-09064f1811e8'
}

const OPT_PIRCE = {
  PUB: 100,
  IPFS: 0.01
}

const PAY_URL = `https://mixin.one/pay?recipient=${BOT.clientId}&`

const urls = {
  me: () => {
    return `${serverHost}/me`
  },
  getUser: () => {
    return `${serverHost}/user/:id`
  },
  auth: () => {
    return `${serverHost}/auth`
  },
  getToken: () => {
    return `${mixinHost}/oauth/token`
  },
  verify: () => {
    return `${serverHost}/verify/:id`
  },
  assets: () => {
    return `${serverHost}/assets`
  },
  asset: (assetId) => {
    return `${mixinHost}/assets/:id`
  },
  getPosts: () => {
    return `${serverHost}/posts`
  },
  posts: () => {
    return `${serverHost}/posts/:id`
  },
  getDrafts: () => {
    return `${serverHost}/drafts`
  },
  drafts: () => {
    return `${serverHost}/drafts/:id`
  },
  post: () => {
    return `${serverHost}/post`
  },
  myPost: () => {
    return `${serverHost}/myPosts/:type`
  },
  subscriber: () => {
    return `${serverHost}/subscriber/:id`
  },
  notify: () => {
    return `${serverHost}/notify/:id`
  }
}

export {
  I18N_OPT,
  urls,
  BOT,
  ASSETS,
  OPT_PIRCE,
  PAY_URL,
  IPFS_NODE,
  IPFS_GATEWAY
}
