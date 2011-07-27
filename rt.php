<?php
class RT {
	
	var $domain = "";
	var $api = "/api/publish";
	
	function __construct($domain) {
		if($domain) $this->domain = $domain;
	}
	
	function publish($channel,$msg) {
	
		$ch = curl_init();
		curl_setopt($ch, CURLOPT_URL,$this->domain.$this->api);
		curl_setopt($ch, CURLOPT_RETURNTRANSFER,1);
		curl_setopt($ch, CURLOPT_POST, 1);
		
		$json = json_encode(array(
			"type"=>"message",
			"channel"=>$channel,
			"identity"=>"sean",
			"data"=>array(
				"msg"=>$msg
			)
		));
		
		curl_setopt($ch,CURLOPT_POSTFIELDS,"&data=$json");
		return curl_exec($ch);
	}
}
?>