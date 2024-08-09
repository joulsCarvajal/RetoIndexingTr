import { createApp } from 'vue'
import './style.css'
import App from './App.vue'

import { OhVueIcon, addIcons } from "oh-vue-icons";
import * as FaIcons from "oh-vue-icons/icons/fa";

import VueSweetalert2 from 'vue-sweetalert2';
import 'sweetalert2/dist/sweetalert2.min.css';


const Fa = Object.values({ ...FaIcons });
addIcons(...Fa);



const app = createApp(App)
app.component("v-icon", OhVueIcon);
app.use(VueSweetalert2);
app.mount("#app");
