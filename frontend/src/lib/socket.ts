import { io } from 'socket.io-client'

const socket = io('/', {
  withCredentials: true,
  autoConnect: false,
  auth: (cb) => {
    cb({ token: localStorage.getItem('jellystics-token') })
  },
})

export default socket
