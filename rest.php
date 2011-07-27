<?php

require "rt.php";
$rt = new RT("http://184.106.226.97:8001");
$response = $rt->publish("chat_advanced","from php");
var_dump($response);

?>