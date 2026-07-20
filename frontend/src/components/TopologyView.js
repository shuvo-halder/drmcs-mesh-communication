import React, { useEffect, useState } from 'react';
import axios from 'axios';

const ANALYTICS_API = 'http://localhost:8090';

const styles = {
  container: {
    background: '#16213e',
    borderRadius: '10px',
    padding: '20px',
    border: '1px solid #0f3460',
    textAlign: 'center'
  },
  title: {
    fontSize: '16px',
    fontWeight: '600',
    color: '#8899aa',
    marginBottom: '15px',
    textTransform: 'uppercase',
    letterSpacing: '1px',
  },
  img: {
    maxWidth: '100%',
    height: 'auto',
    borderRadius: '5px',
  },
  placeholder: {
    padding: '40px',
    color: '#8899aa',
  },
};

export default function TopologyView() {
  const [image, setImage] = useState('');

  useEffect(() => {
    let mounted = true;
    const fetchImage = async () => {
      try {
        const res = await axios.get(`${ANALYTICS_API}/api/v2/topology/image`);
        if (mounted && res.data.image) {
          setImage(`data:image/png;base64,${res.data.image}`);
        }
      } catch (err) {
        console.error('Failed to fetch topology:', err);
      }
    };
    fetchImage();
    const interval = setInterval(fetchImage, 15000);
    return () => { mounted = false; clearInterval(interval); };
  }, []);

  return (
    <div style={styles.container}>
      <div style={styles.title}>Network Topology Visualization</div>
      {image ? (
        <img src={image} alt="Network Topology" style={styles.img} />
      ) : (
        <div style={styles.placeholder}>
          <p>Loading topology visualization...</p>
          <p style={{ fontSize: '12px', marginTop: '10px' }}>
            Ensure the analytics server is running and nodes are connected
          </p>
        </div>
      )}
    </div>
  );
}