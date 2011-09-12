<?php

require "rt.php";
$rt = new RT("http://localhost:8001");
$response = $rt->publish("chat_advanced","from php");
if($rt->status == 200) {
	echo "success";
} else {
	var_dump($response);
}	

?>