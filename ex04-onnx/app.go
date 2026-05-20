package main

import (
	// "bytes"
	"database/sql"
	"encoding/json"
	"fmt"

	// "io"
	"log"
	// "net/http"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type InspectionData struct {
	ID     int    `json:"-"` // ใช้ภายใน ไม่ต้องส่งไป JSON
	LotID  string `json:"lot_code"`
	Seq    int    `json:"duck_sequence"`
	Date   string `json:"inspection_date"`
	IsD1   bool   `json:"wing_left"`
	IsD2   bool   `json:"wing_right"`
	IsD3   bool   `json:"back"`
	IsD4   bool   `json:"leg_left"`
	IsD5   bool   `json:"leg_right"`
	ParamA int    `json:"scratch"`
	ParamB int    `json:"bruise"`
}

type Payload struct {
	DeviceID string           `json:"device_id"`
	Records  []InspectionData `json:"records"`
}

func main() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=admin password=admin dbname=duck_management sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for {
		start := time.Now()
		rows, err := db.Query("SELECT id, lot_code, duck_sequence, inspection_date, wing_left, wing_right, back, leg_left, leg_right , scratch, bruise FROM factory_duck_defects WHERE is_sync = false")
		if err != nil {
			log.Println("Query error:", err)
			time.Sleep(10 * time.Second)
			continue
		}

		var records []InspectionData
		var syncedIDs []string

		for rows.Next() {
			var r InspectionData
			err := rows.Scan(&r.ID, &r.LotID, &r.Seq, &r.Date, &r.IsD1, &r.IsD2, &r.IsD3, &r.IsD4, &r.IsD5, &r.ParamA, &r.ParamB)
			if err == nil {
				records = append(records, r)
				syncedIDs = append(syncedIDs, fmt.Sprintf("%d", r.ID))
			}
		}
		rows.Close()

		if len(records) > 0 {
			// 2. ส่งไป Online
			success := sendToOnline(records)

			// 3. ถ้าส่งสำเร็จ (Status 200) ให้เปลี่ยน is_synced เป็น 1 (True) ใน Local DB
			if success {
				idList := strings.Join(syncedIDs, ",")
				query := fmt.Sprintf("UPDATE factory_duck_defects SET is_sync = true WHERE id IN (%s)", idList)
				_, err := db.Exec(query)
				if err != nil {
					log.Println("Update error:", err)
				} else {
					fmt.Printf("✅ Updated %d rows to is_sync = true\n", len(records))
				}
				fmt.Printf("✅ Synced %d rows to online\n", len(records))
			}
		}

		duration := time.Since(start)
		fmt.Printf("⏱️ เวลาที่ใช้ทั้งหมด: %v\n", duration)
		time.Sleep(10 * time.Minute)
	}
}

func sendToOnline(data []InspectionData) bool {
	payload := Payload{DeviceID: "CAPTAIN-NODE-01", Records: data}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Println("JSON Marshal error:", err)
		return false
	}

	// คำนวณขนาด
	byteSize := len(jsonData)
	bitSize := byteSize * 8
	kbSize := float64(byteSize) / 1024

	fmt.Printf("📤 JSON Payload:\n%s\n", jsonData)
	fmt.Println("---------------------------------")
	fmt.Printf("📊 Data Size Info:\n")
	fmt.Printf("- Size in Bytes: %d B\n", byteSize)
	fmt.Printf("- Size in Bits:  %d bit\n", bitSize)
	fmt.Printf("- Size in KB:    %.2f KB\n", kbSize)
	fmt.Println("---------------------------------")

	// // ส่ง HTTP POST ไป online server
	// const endpoint = "https://your-online-api.com/api/factory_duck_defects" // เปลี่ยนเป็น URL จริงของคุณที่นี่
	// req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	// if err != nil {
	// 	log.Println("HTTP NewRequest error:", err)
	// 	return false
	// }
	// req.Header.Set("Content-Type", "application/json")

	// client := &http.Client{Timeout: 30 * time.Second}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	log.Println("HTTP Do error:", err)
	// 	return false
	// }
	// defer resp.Body.Close()

	// respBody, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Println("Read response body error:", err)
	// 	return false
	// }

	// if resp.StatusCode >= 200 && resp.StatusCode < 300 {
	// 	fmt.Printf("✅ ส่งข้อมูลสำเร็จ! Status: %d, Response: %s\n", resp.StatusCode, string(respBody))
	// 	return true
	// } else {
	// 	fmt.Printf("❌ ส่งข้อมูลล้มเหลว! Status: %d, Response: %s\n", resp.StatusCode, string(respBody))
	// 	return false
	// }
	return true
}
