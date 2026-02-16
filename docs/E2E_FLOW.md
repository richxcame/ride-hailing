# End-to-End Ride-Hailing Flow

## Overview
This document describes the complete flow for drivers and riders in the ride-hailing system, from registration to ride completion.

---

## 🚗 Driver Flow

### 1. Registration & Onboarding
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Registration                                         │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/auth/register
{
  "email": "driver@example.com",
  "password": "SecurePass123!",
  "phone_number": "+1234567890",
  "first_name": "John",
  "last_name": "Driver",
  "role": "driver"
}

✅ System automatically creates:
   - User account (users table)
   - Driver profile with PENDING- prefixes (drivers table)
   - Returns JWT token

Response:
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "driver@example.com",
    "role": "driver",
    ...
  }
}
```

### 2. Complete Profile (Update Vehicle Info)
```
┌─────────────────────────────────────────────────────────────┐
│ Update Driver Profile (Optional - Can be done via Admin)   │
└─────────────────────────────────────────────────────────────┘

PUT /api/v1/auth/profile
Authorization: Bearer {driver_token}
{
  "license_number": "DL-123456",
  "vehicle_model": "Toyota Camry 2020",
  "vehicle_plate": "ABC-1234",
  "vehicle_color": "Silver",
  "vehicle_year": 2020
}
```

### 3. Go Online (Set Status Available)
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Goes Online                                          │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/driver/status
Authorization: Bearer {driver_token}
{
  "status": "available"
}

✅ System:
   - Validates eligibility (MVP: always passes)
   - Tracks session start time in Redis
   - Updates driver status to "available"

Response:
{
  "success": true,
  "data": {
    "status": "available",
    "updated_at": "2024-02-15T10:30:00Z"
  }
}
```

### 4. Update Location (Continuous)
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Location Updates (Every 5-10 seconds)               │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/geo/location
Authorization: Bearer {driver_token}
{
  "latitude": 40.7128,
  "longitude": -74.0060,
  "heading": 45.0,    // Direction in degrees (0-360)
  "speed": 25.5       // Speed in km/h
}

✅ System:
   - Stores location in Redis (key: driver:location:{id})
   - Updates geo-spatial index (GEORADIUS queries)
   - Updates H3 cell assignment for efficient searching
   - Sets 24-hour expiry (auto-cleanup)

Response:
{
  "success": true,
  "data": {
    "message": "location updated"
  }
}

⚠️ Driver should send this every 5-10 seconds while online
```

### 5. Receive Ride Offer (WebSocket)
```
┌─────────────────────────────────────────────────────────────┐
│ Receive Ride Offer via WebSocket                           │
└─────────────────────────────────────────────────────────────┘

WebSocket Connection:
ws://localhost:8086/ws?token={driver_token}

Incoming Message:
{
  "type": "ride.offer",
  "ride_id": "ride-uuid",
  "data": {
    "rider_name": "Jane Rider",
    "rider_rating": 4.8,
    "pickup_location": {
      "latitude": 40.7580,
      "longitude": -73.9855,
      "address": "123 Main St, New York, NY"
    },
    "dropoff_location": {
      "latitude": 40.7489,
      "longitude": -73.9680,
      "address": "456 Park Ave, New York, NY"
    },
    "ride_type": "Economy",
    "estimated_fare": 25.50,
    "estimated_distance": 5.2,
    "estimated_duration": 15,
    "distance_to_pickup": 2.1,
    "eta_to_pickup": 7,
    "currency": "USD",
    "expires_at": "2024-02-15T10:31:00Z",
    "timeout_seconds": 30
  }
}

⏰ Driver has 30 seconds to accept/reject
```

### 6. Accept Ride
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Accepts Ride                                         │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/rides/{ride_id}/accept
Authorization: Bearer {driver_token}

✅ System:
   - Updates ride status: pending → accepted
   - Assigns driver_id to ride
   - Changes driver status: available → busy
   - Publishes ride.accepted event
   - Notifies rider via WebSocket
   - Cancels pending offers to other drivers

Response:
{
  "success": true,
  "data": {
    "ride_id": "ride-uuid",
    "status": "accepted",
    "rider": {...},
    "pickup_location": {...},
    "dropoff_location": {...}
  }
}
```

### 7. Navigate to Pickup
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Navigates to Pickup (Continue Location Updates)     │
└─────────────────────────────────────────────────────────────┘

Continue sending location updates:
POST /api/v1/geo/location
(every 5-10 seconds)

Rider sees driver approaching in real-time via WebSocket
```

### 8. Arrive at Pickup
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Arrives at Pickup Location                          │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/rides/{ride_id}/arrive
Authorization: Bearer {driver_token}

✅ System:
   - Updates ride status: accepted → arrived
   - Notifies rider via WebSocket
   - Starts waiting time counter

Response:
{
  "success": true,
  "data": {
    "status": "arrived",
    "arrived_at": "2024-02-15T10:35:00Z"
  }
}
```

### 9. Start Ride
```
┌─────────────────────────────────────────────────────────────┐
│ Rider Gets In, Driver Starts Ride                          │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/rides/{ride_id}/start
Authorization: Bearer {driver_token}

✅ System:
   - Updates ride status: arrived → in_progress
   - Records start location & time
   - Publishes ride.started event
   - Starts tracking actual distance/duration

Response:
{
  "success": true,
  "data": {
    "status": "in_progress",
    "started_at": "2024-02-15T10:37:00Z"
  }
}
```

### 10. Complete Ride
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Completes Ride at Destination                       │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/rides/{ride_id}/complete
Authorization: Bearer {driver_token}

✅ System:
   - Updates ride status: in_progress → completed
   - Calculates final fare based on actual distance/duration
   - Processes payment
   - Updates driver/rider statistics
   - Changes driver status: busy → available
   - Publishes ride.completed event

Response:
{
  "success": true,
  "data": {
    "status": "completed",
    "completed_at": "2024-02-15T10:52:00Z",
    "final_fare": 27.50,
    "actual_distance": 5.5,
    "actual_duration": 15,
    "currency": "USD"
  }
}
```

### 11. Rate Rider (Optional)
```
POST /api/v1/rides/{ride_id}/rate
Authorization: Bearer {driver_token}
{
  "rating": 5,
  "comment": "Great passenger!"
}
```

### 12. Go Offline
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Goes Offline (End of Shift)                         │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/driver/status
Authorization: Bearer {driver_token}
{
  "status": "offline"
}

✅ System:
   - Updates status to offline
   - Ends session tracking
   - Returns session summary
   - Removes from geo-spatial index

Response:
{
  "success": true,
  "data": {
    "status": "offline",
    "session_summary": {
      "online_duration_minutes": 240,
      "started_at": "2024-02-15T06:00:00Z",
      "ended_at": "2024-02-15T10:00:00Z"
    }
  }
}
```

---

## 👤 Rider Flow

### 1. Registration
```
┌─────────────────────────────────────────────────────────────┐
│ Rider Registration                                          │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/auth/register
{
  "email": "rider@example.com",
  "password": "SecurePass123!",
  "phone_number": "+0987654321",
  "first_name": "Jane",
  "last_name": "Rider",
  "role": "rider"
}

Response:
{
  "success": true,
  "data": {
    "id": "rider-uuid",
    "email": "rider@example.com",
    "role": "rider",
    ...
  }
}
```

### 2. Get Fare Estimate
```
┌─────────────────────────────────────────────────────────────┐
│ Get Fare Estimate Before Requesting Ride                   │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/rides/estimate
Authorization: Bearer {rider_token}
{
  "pickup_latitude": 40.7128,
  "pickup_longitude": -74.0060,
  "dropoff_latitude": 40.7580,
  "dropoff_longitude": -73.9855,
  "ride_type_id": "economy-uuid"  // Optional
}

Response:
{
  "success": true,
  "data": {
    "estimated_fare": 25.50,
    "estimated_distance": 5.2,
    "estimated_duration": 15,
    "currency": "USD",
    "breakdown": {
      "base_fare": 5.00,
      "distance_fare": 15.00,
      "time_fare": 5.50
    }
  }
}
```

### 3. Request Ride
```
┌─────────────────────────────────────────────────────────────┐
│ Request a Ride                                              │
└─────────────────────────────────────────────────────────────┘

POST /api/v1/rides
Authorization: Bearer {rider_token}
{
  "pickup_latitude": 40.7128,
  "pickup_longitude": -74.0060,
  "pickup_address": "123 Main St, New York, NY",
  "dropoff_latitude": 40.7580,
  "dropoff_longitude": -73.9855,
  "dropoff_address": "456 Park Ave, New York, NY",
  "ride_type_id": "economy-uuid",
  "payment_method_id": "payment-uuid"
}

✅ System:
   - Creates ride with status "pending"
   - Calculates fare estimate
   - Publishes ride.requested event (NATS)
   - Matching service finds nearby drivers
   - Sends offers to 3-5 closest drivers via WebSocket
   - If no acceptance, expands search radius

Response:
{
  "success": true,
  "data": {
    "id": "ride-uuid",
    "status": "pending",
    "estimated_fare": 25.50,
    "estimated_duration": 15,
    "created_at": "2024-02-15T10:30:00Z"
  }
}
```

### 4. Wait for Driver Acceptance (WebSocket)
```
┌─────────────────────────────────────────────────────────────┐
│ Receive Driver Assignment via WebSocket                    │
└─────────────────────────────────────────────────────────────┘

WebSocket Connection:
ws://localhost:8086/ws?token={rider_token}

Incoming Messages:

1. Searching for drivers:
{
  "type": "ride.searching",
  "ride_id": "ride-uuid",
  "data": {
    "status": "searching",
    "drivers_notified": 5
  }
}

2. Driver accepted:
{
  "type": "ride.accepted",
  "ride_id": "ride-uuid",
  "data": {
    "status": "accepted",
    "driver": {
      "id": "driver-uuid",
      "name": "John Driver",
      "rating": 4.9,
      "vehicle_model": "Toyota Camry 2020",
      "vehicle_plate": "ABC-1234",
      "vehicle_color": "Silver",
      "phone_number": "+1234567890"
    },
    "eta_to_pickup": 7
  }
}

3. Driver location updates (real-time):
{
  "type": "driver.location",
  "ride_id": "ride-uuid",
  "data": {
    "latitude": 40.7150,
    "longitude": -74.0070,
    "heading": 45.0,
    "distance_to_pickup": 1.5,
    "eta_to_pickup": 5
  }
}
```

### 5. Driver Arrives
```
┌─────────────────────────────────────────────────────────────┐
│ Driver Arrives at Pickup                                   │
└─────────────────────────────────────────────────────────────┘

WebSocket Message:
{
  "type": "ride.driver_arrived",
  "ride_id": "ride-uuid",
  "data": {
    "status": "arrived",
    "arrived_at": "2024-02-15T10:35:00Z"
  }
}

Rider receives notification to come out
```

### 6. Ride Starts
```
┌─────────────────────────────────────────────────────────────┐
│ Ride Starts                                                 │
└─────────────────────────────────────────────────────────────┘

WebSocket Message:
{
  "type": "ride.started",
  "ride_id": "ride-uuid",
  "data": {
    "status": "in_progress",
    "started_at": "2024-02-15T10:37:00Z"
  }
}

Rider can track progress in real-time via map
```

### 7. Track Ride Progress
```
GET /api/v1/rides/{ride_id}
Authorization: Bearer {rider_token}

Response:
{
  "success": true,
  "data": {
    "id": "ride-uuid",
    "status": "in_progress",
    "driver": {...},
    "pickup_location": {...},
    "dropoff_location": {...},
    "current_location": {
      "latitude": 40.7489,
      "longitude": -73.9680
    },
    "started_at": "2024-02-15T10:37:00Z",
    "estimated_arrival": "2024-02-15T10:52:00Z"
  }
}
```

### 8. Ride Completes
```
┌─────────────────────────────────────────────────────────────┐
│ Ride Completed                                              │
└─────────────────────────────────────────────────────────────┘

WebSocket Message:
{
  "type": "ride.completed",
  "ride_id": "ride-uuid",
  "data": {
    "status": "completed",
    "completed_at": "2024-02-15T10:52:00Z",
    "final_fare": 27.50,
    "actual_distance": 5.5,
    "actual_duration": 15,
    "payment_status": "processed"
  }
}
```

### 9. Rate Driver
```
POST /api/v1/rides/{ride_id}/rate
Authorization: Bearer {rider_token}
{
  "rating": 5,
  "comment": "Excellent driver!"
}
```

### 10. View Ride History
```
GET /api/v1/rides/history?limit=20&offset=0
Authorization: Bearer {rider_token}

Response:
{
  "success": true,
  "data": [
    {
      "id": "ride-uuid",
      "status": "completed",
      "driver": {...},
      "fare": 27.50,
      "distance": 5.5,
      "completed_at": "2024-02-15T10:52:00Z"
    },
    ...
  ]
}
```

---

## 🔄 Cancel Ride Flow

### Rider Cancels Before Acceptance
```
DELETE /api/v1/rides/{ride_id}
Authorization: Bearer {rider_token}

✅ System:
   - Updates status to "cancelled"
   - Cancels pending driver offers
   - No cancellation fee
```

### Rider Cancels After Acceptance
```
POST /api/v1/rides/{ride_id}/cancel
Authorization: Bearer {rider_token}
{
  "reason": "Changed my mind"
}

✅ System:
   - Updates status to "cancelled"
   - May charge cancellation fee
   - Notifies driver
   - Driver becomes available again
```

### Driver Cancels
```
POST /api/v1/rides/{ride_id}/cancel
Authorization: Bearer {driver_token}
{
  "reason": "Emergency"
}

✅ System:
   - Updates status to "cancelled"
   - Searches for new driver
   - May affect driver's acceptance rate
```

---

## 🔑 Key System Events

### NATS Event Flow
```
ride.requested
  ↓
matching service finds drivers
  ↓
sends WebSocket offers to drivers
  ↓
ride.accepted (one driver)
  ↓
cancels other pending offers
  ↓
ride.started
  ↓
ride.completed
  ↓
payment.processed
```

### Redis Keys
```
driver:location:{driver_id}     → Driver's current location
driver:status:{driver_id}       → available/busy/offline
driver:session:{driver_id}      → Session tracking data
drivers:geo                     → GEO spatial index
h3:{cell_id}                    → H3 cell → driver mapping
ride_status:{ride_id}           → Ride status cache
ride_offer:{ride_id}:{driver_id} → Offer tracking
```

---

## ⚠️ Important Notes

1. **Location Updates**: Drivers must update location every 5-10 seconds
2. **WebSocket**: Both drivers and riders must maintain WebSocket connection for real-time updates
3. **Timeouts**: Ride offers expire after 30 seconds
4. **Retry Logic**: If no driver accepts, system automatically expands search radius
5. **Status Management**: Driver status (available/busy/offline) is critical for matching
6. **Session Tracking**: Redis TTL ensures automatic cleanup of stale data

---

## 🎯 Next Steps

1. Test driver registration → going online → location updates
2. Test rider registration → requesting ride
3. Verify WebSocket connections work for both roles
4. Test complete ride flow end-to-end
5. Test edge cases (cancellations, timeouts, no drivers available)
